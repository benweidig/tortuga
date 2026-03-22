package ui

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/muesli/termenv"
	"golang.org/x/term"

	"github.com/benweidig/tortuga/internal/model"
	"github.com/benweidig/tortuga/internal/runner"
)

type Action int

const (
	ActionSync     Action = iota
	ActionIncoming        // pull only, no push
	ActionNone
)

// FilterRepos returns only the repos that need work for the given action.
func FilterRepos(repos []model.Repo, action Action) []model.Repo {
	var filtered []model.Repo
	for _, r := range repos {
		switch action {
		case ActionSync:
			if r.Ahead > 0 || r.Behind > 0 {
				filtered = append(filtered, r)
			}
		case ActionIncoming:
			if r.Behind > 0 {
				filtered = append(filtered, r)
			}
		}
	}
	return filtered
}

// AskAction reads keypresses from the tty and returns the chosen action.
// The terminal is put into raw mode so no Enter is required.
// Pressing '?' shows an explanation and re-renders the prompt.
func AskAction(tty *os.File, repos []model.Repo) (Action, error) {
	summary := buildPromptSummary(repos)

	ttyOut := termenv.NewOutput(tty)
	prompt := ">>> Sync Changes?"
	if summary != "" {
		prompt += " " + ttyOut.String(summary).Foreground(termenv.ANSIYellow).String()
	}
	prompt += " [Y/n/i/?]"

	// Blank line between the status table and the prompt.
	fmt.Fprintln(tty)
	fmt.Fprint(tty, prompt)

	oldState, err := term.MakeRaw(int(tty.Fd()))
	if err != nil {
		return ActionNone, fmt.Errorf("raw mode: %w", err)
	}
	defer term.Restore(int(tty.Fd()), oldState)

	buf := make([]byte, 1)
	for {
		if _, err := tty.Read(buf); err != nil {
			return ActionNone, err
		}

		switch buf[0] {
		case 'y', 'Y', '\r', '\n':
			fmt.Fprint(tty, "\r\n")
			return ActionSync, nil
		case 'n', 'N':
			fmt.Fprint(tty, "\r\n")
			return ActionNone, nil
		case 'i', 'I':
			fmt.Fprint(tty, "\r\n")
			return ActionIncoming, nil
		case '?':
			// Clear from start of prompt line to end of screen, show explanation,
			// then re-render the prompt on the same line.
			fmt.Fprint(tty, "\r\033[J")
			fmt.Fprint(tty, "Available options:\r\n")
			fmt.Fprint(tty, "  y = Full Sync (stash, pull+rebase, push) [default]\r\n")
			fmt.Fprint(tty, "  n = No sync at all\r\n")
			fmt.Fprint(tty, "  i = Sync incoming only (stash, pull+rebase)\r\n")
			fmt.Fprint(tty, "  ? = Explain options\r\n")
			fmt.Fprint(tty, "\r\n")
			fmt.Fprint(tty, prompt)
		case '\x03', '\x1b': // Ctrl-C, Escape
			fmt.Fprint(tty, "\r\n")
			return ActionNone, nil
		}
	}
}

// buildPromptSummary returns a compact summary of total ahead/behind commits
// across all repos, e.g. "5↑ 8↓". Empty string if nothing pending.
func buildPromptSummary(repos []model.Repo) string {
	var totalAhead, totalBehind int
	for _, r := range repos {
		totalAhead += r.Ahead
		totalBehind += r.Behind
	}
	var parts []string
	if totalAhead > 0 {
		parts = append(parts, fmt.Sprintf("%d↑", totalAhead))
	}
	if totalBehind > 0 {
		parts = append(parts, fmt.Sprintf("%d↓", totalBehind))
	}
	return strings.Join(parts, " ")
}

func RunSyncPhase(repos []model.Repo, action Action, sem chan struct{}, renderer *Renderer) {
	// mu guards both repos and all renderer calls, same reasoning as RunFetchPhase.
	var mu sync.Mutex
	spinnerIdx := 0

	done := make(chan struct{})
	var spinnerWg sync.WaitGroup
	spinnerWg.Go(func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				mu.Lock()
				spinnerIdx = (spinnerIdx + 1) % len(spinnerFrames)
				renderer.RenderTick(repos, spinnerFrames[spinnerIdx])
				mu.Unlock()
			case <-done:
				return
			}
		}
	})

	renderer.Render(repos, spinnerFrames[0])

	// runner.RunSyncPhase blocks until all repos are done and calls the callback
	// serially, so the callback itself needs no additional synchronisation beyond
	// the shared mu.
	runner.RunSyncPhase(repos, action == ActionSync, sem, func(idx int, repo model.Repo) {
		mu.Lock()
		repos[idx] = repo
		renderer.RenderRowUpdate(idx, repos, spinnerFrames[spinnerIdx])
		mu.Unlock()
	})

	close(done)
	// Must wait for the spinner goroutine to exit before the final render.
	spinnerWg.Wait()

	renderer.Render(repos, "")
	renderer.RenderErrors(repos)
}
