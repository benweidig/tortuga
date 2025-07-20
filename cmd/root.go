package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

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

	manager := repo.NewManager()
	err := manager.Discover(basePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error discovering repositories: '%s'.\n", err)
		os.Exit(1)
	}

	if manager.Count() == 0 {
		fmt.Fprintf(os.Stderr, "No repositories found at '%s'.\n", basePath)
		os.Exit(1)
	}

	// /////////////////////////////////////////////////////////////////////////
	// Step 3: Update repositories
	// /////////////////////////////////////////////////////////////////////////

	fmt.Println()

	// Start live writer which we will use throughout the rendering
	w := ui.NewStdoutWriter()

	updateRepositories(manager, w)

	// /////////////////////////////////////////////////////////////////////////
	// Step 4: Check if we can sync at all
	// /////////////////////////////////////////////////////////////////////////

	if !manager.HasChangesToSync() {
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
			if manager.TotalIncoming() > 0 {
				prompt += gchalk.WithBrightYellow().Sprintf(" %d↓", manager.TotalIncoming())
			}

			if manager.TotalOutgoing() > 0 {
				prompt += gchalk.WithBrightYellow().Sprintf(" %d↑", manager.TotalOutgoing())
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

	syncRepositories(manager, syncIncomingOnly, w)

	fmt.Println()
}

func updateRepositories(manager repo.RepositoryManager, w *ui.StdoutWriter) {
	// Initial output showing all repos
	repos := manager.GetRepositories()
	w.Render(func() {
		ui.WriteRepositoryStatus(w, repos, false)
	})

	// Update all repositories with progress callback
	ctx := context.Background()
	manager.UpdateAll(ctx, func() {
		repos := manager.GetRepositories()
		w.Render(func() {
			ui.WriteRepositoryStatus(w, repos, false)
		})
	})
}

func syncRepositories(manager repo.RepositoryManager, incomingOnly bool, w *ui.StdoutWriter) {
	// Reset live writer and render the repositories
	w.Reset()
	repos := manager.GetRepositories()
	ui.WriteRepositoryStatus(w, repos, incomingOnly)

	// Sync all repositories with progress callback
	ctx := context.Background()
	manager.SyncAll(ctx, incomingOnly, func() {
		repos := manager.GetRepositories()
		w.Render(func() {
			ui.WriteRepositoryStatus(w, repos, incomingOnly)
		})
	})
}
