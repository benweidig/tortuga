package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/benweidig/tortuga/git"
	"github.com/benweidig/tortuga/repo"
	"github.com/benweidig/tortuga/ui"
	"github.com/benweidig/tortuga/version"
	"github.com/jwalton/gchalk"

	"github.com/spf13/cobra"
)

// Arguments of the command
var (
	monochromeArg bool
	yesArg        bool
)

// RootCmd is the only command, so this is Tortuga
var RootCmd = &cobra.Command{
	Version: version.BuildVersion(),
	Use:     "tt",
	Short:   "Tortuga",
	Args:    cobra.MaximumNArgs(1),
	Long:    "CLI tool for fetching/rebasing multiple git repositories at once",
	Run:     runCommand,
}

func init() {
	RootCmd.Flags().BoolVarP(&monochromeArg, "monochrome", "m", false, "Monochrome output, no ANSI colorize")
	RootCmd.Flags().BoolVarP(&yesArg, "yes", "y", false, "Anwser 'Yes' to 'sync' prompt")
}

func runCommand(_ *cobra.Command, args []string) {

	// /////////////////////////////////////////////////////////////////////////
	// Step 1: Parse arguments and prepare requirements
	// /////////////////////////////////////////////////////////////////////////

	// Determinate the directory to check.
	var basePath string

	// There can only be 0 or 1 arguments, so this check is enough
	if len(args) == 1 {
		basePath = args[0]
	} else {
		// Falback to actual working directory
		wd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Couldn't retrieve working directory: '%s'.\n", err)
			os.Exit(1)
		}
		basePath = wd
	}

	// Disable colors if requested either via arg or env, see http://no-color.org/.
	// The color library might disable color nontheless if it thinks the terminal isn't
	// supporting it.
	_, noColorEnvExists := os.LookupEnv("NO_COLOR")
	monochromeArg = monochromeArg || noColorEnvExists
	if monochromeArg {
		gchalk.SetLevel(gchalk.LevelNone)
	}

	// /////////////////////////////////////////////////////////////////////////
	// Step 2: Find repositories
	// /////////////////////////////////////////////////////////////////////////

	repos, _ := findRepositories(basePath)

	if len(repos) == 0 {
		fmt.Fprintf(os.Stderr, "No repositories found at '%s'.\n", basePath)
		os.Exit(1)
	}

	// /////////////////////////////////////////////////////////////////////////
	// Step 3: Update repositories
	// /////////////////////////////////////////////////////////////////////////

	fmt.Println()

	// Start live writer which we will use throughout the rendering
	w := ui.NewStdoutWriter()

	updateRepositories(repos, w)

	// /////////////////////////////////////////////////////////////////////////
	// Step 4: Check if we can sync at all
	// /////////////////////////////////////////////////////////////////////////

	incoming := 0
	outgoing := 0

	for _, r := range repos {
		incoming += r.Incoming
		outgoing += r.Outgoing
	}

	if incoming == 0 && outgoing == 0 {
		os.Exit(0)
	}

	// /////////////////////////////////////////////////////////////////////////
	// Step 5a: Ask if you should sync
	// /////////////////////////////////////////////////////////////////////////

	var syncIncomingOnly bool

	if !yesArg {

		// Mark the current position so we can reset properly
		w.Mark()

		for {
			// Flush first, or we need to flush after each write
			w.Flush()

			prompt := ""
			if incoming > 0 {
				prompt += gchalk.WithBrightYellow().Sprintf(" %d↓", incoming)
			}

			if outgoing > 0 {
				prompt += gchalk.WithBrightYellow().Sprintf(" %d↑", outgoing)
			}

			fmt.Fprintf(w, "%s Sync Changes?%s [Y/n/i/?] ", gchalk.WithWhite().Bold(">>>"), prompt)
			w.Flush()

			r := bufio.NewReader(os.Stdin)

			answer, err := r.ReadString('\n')
			if err != nil {
				fmt.Fprintf(os.Stderr, "Couldn't get prompt answer: '%s'.\n", err)
				os.Exit(1)
			}

			w.AddLineBreaks(1)

			// Sanitize
			answer = strings.TrimSpace(strings.ToLower(answer))
			if len(answer) > 1 {
				fmt.Fprintf(w, "Invalid option: '%s'\n\n", answer)
				continue
			}

			if answer == "n" {
				os.Exit(0)
			} else if answer == "i" {
				syncIncomingOnly = true
				break
			} else if answer == "?" {
				w.ResetToMarker()

				fmt.Fprintln(w, gchalk.Bold("Available options:"))
				fmt.Fprintf(w, "  %s = %s", gchalk.Bold("y"), "Full Sync (stash, pull+rebase, push) [default]\n")
				fmt.Fprintf(w, "  %s = %s", gchalk.Bold("n"), "No sync at all\n")
				fmt.Fprintf(w, "  %s = %s", gchalk.Bold("i"), "Sync incoming only (stash, pull+rebase)\n")
				fmt.Fprintf(w, "  %s = %s", gchalk.Bold("?"), "Explain options\n")
				fmt.Fprintln(w)
			} else if answer == "y" || answer == "" {
				break
			}
		}
	}

	fmt.Fprintln(w)

	// /////////////////////////////////////////////////////////////////////////
	// Step 5b: Do the actual sync
	// /////////////////////////////////////////////////////////////////////////

	syncRepositories(repos, syncIncomingOnly, w)

	fmt.Println()
}

func findRepositories(basePath string) ([]*repo.Repository, error) {
	var repos []*repo.Repository

	if git.IsRepo(basePath) {
		r, err := repo.NewRepository(basePath)
		repos = append(repos, r)
		return repos, err
	}

	entries, err := os.ReadDir(basePath)
	if err != nil {
		return repos, err
	}

	for _, entry := range entries {
		// We are only interested in directories
		if !entry.IsDir() {
			continue
		}

		// Build paths and check if we got .git directory
		entryPath := path.Join(basePath, entry.Name())
		if !git.IsRepo(entryPath) {
			continue
		}

		// Build repository. We ignore errors so all will be displayed
		r, _ := repo.NewRepository(entryPath)
		repos = append(repos, r)
	}

	return repos, nil
}

func updateRepositories(repos []*repo.Repository, w *ui.StdoutWriter) {

	// 2. Initial output showing all repos
	w.Render(func() {
		ui.WriteRepositoryStatus(w, repos, false)
	})

	// 3. Start a waitgroup
	var wg sync.WaitGroup
	wg.Add(len(repos))

	// 4. Iterate over the repos and parallel check/update the repos and update the output
	for idx := range repos {
		r := repos[idx]
		go func() {
			r.Update()

			w.Render(func() {
				ui.WriteRepositoryStatus(w, repos, false)
			})

			wg.Done()
		}()
	}

	// 5. Wait for all goroutines to finish
	wg.Wait()
}

func syncRepositories(repos []*repo.Repository, incomingOnly bool, w *ui.StdoutWriter) {
	for idx := range repos {
		r := repos[idx]

		// No need to check an unsafe repository
		if r.State == repo.StateError {
			continue
		}

		if r.NeedsSync() {
			r.State = repo.StateNeedsSync
		} else {
			r.State = repo.StateNoSyncNeeded
		}
	}

	// 2. Reset live writer and render the repositories
	w.Reset()
	ui.WriteRepositoryStatus(w, repos, incomingOnly)

	// 3. Do the work async for better speed
	var wg sync.WaitGroup
	wg.Add(len(repos))

	for idx := range repos {
		r := repos[idx]
		if r.State != repo.StateNeedsSync {
			w.Render(func() {
				ui.WriteRepositoryStatus(w, repos, incomingOnly)
			})
			wg.Done()
			continue
		}

		go func() {
			r.Sync(incomingOnly)

			w.Render(func() {
				ui.WriteRepositoryStatus(w, repos, incomingOnly)
			})

			wg.Done()
		}()
	}
	wg.Wait()
}
