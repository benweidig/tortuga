package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

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
	verboseArg    bool
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
	RootCmd.Flags().BoolVarP(&verboseArg, "verbose", "v", false, "Show verbose error output")
}

func runCommand(_ *cobra.Command, args []string) {
	basePath := determineBasePath(args)
	setupColorOutput()

	manager := discoverRepositories(basePath)
	if manager.Count() == 0 {
		fmt.Fprintf(os.Stderr, "No repositories found at '%s'.\n", basePath)
		os.Exit(1)
	}

	fmt.Println()
	
	w := ui.NewStdoutWriter()
	ctx := context.Background()

	updateRenderer := ui.NewUIRenderer(manager, w)
	updateRenderer.Start(ctx)
	failed := updateRepositories(manager, updateRenderer)
	updateRenderer.Stop() // blocks until final render is done; cursor is now at a stable position

	if !manager.HasChangesToSync() {
		if failed {
			os.Exit(1)
		}
		return
	}

	syncIncomingOnly := promptUserForSyncOptions(manager, w)
	fmt.Fprintln(w)

	syncRenderer := ui.NewUIRenderer(manager, w)
	syncRenderer.Start(ctx)
	failed = syncRepositories(manager, syncIncomingOnly, syncRenderer) || failed
	syncRenderer.Stop()
	fmt.Println()

	if failed {
		os.Exit(1)
	}
}

// determineBasePath returns the target directory from args or current working directory
func determineBasePath(args []string) string {
	if len(args) == 1 {
		return args[0]
	}

	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't retrieve working directory: '%s'.\n", err)
		os.Exit(1)
	}
	return wd
}

// setupColorOutput configures color output based on flags and environment
func setupColorOutput() {
	_, noColorEnvExists := os.LookupEnv("NO_COLOR")
	if monochromeArg || noColorEnvExists {
		gchalk.SetLevel(gchalk.LevelNone)
	}
}

// discoverRepositories finds and initializes all repositories in the given path
func discoverRepositories(basePath string) repo.RepositoryManager {
	manager := repo.NewManager()
	if err := manager.Discover(basePath); err != nil {
		fmt.Fprintf(os.Stderr, "Error discovering repositories: '%s'.\n", err)
		os.Exit(1)
	}
	return manager
}

// promptUserForSyncOptions presents sync options to user and returns their choice
func promptUserForSyncOptions(manager repo.RepositoryManager, w *ui.StdoutWriter) bool {
	if yesArg {
		return false // full sync
	}

	w.Mark()
	for {
		w.Flush()

		prompt := buildSyncPrompt(manager)
		fmt.Fprintf(w, "%s Sync Changes?%s [Y/n/i/?] ", gchalk.WithWhite().Bold(">>>"), prompt)
		w.Flush()

		answer := readUserInput()
		w.AddLineBreaks(1)

		switch processUserAnswer(answer, w) {
		case "n":
			os.Exit(0)
		case "i":
			return true // incoming only
		case "?":
			showHelpOptions(w)
		case "y", "":
			return false // full sync
		default:
			// continue loop for invalid input
		}
	}
}

// buildSyncPrompt creates the sync prompt showing incoming/outgoing counts
func buildSyncPrompt(manager repo.RepositoryManager) string {
	var parts []string
	if manager.TotalIncoming() > 0 {
		parts = append(parts, gchalk.WithBrightYellow().Sprintf(" %d↓", manager.TotalIncoming()))
	}
	if manager.TotalOutgoing() > 0 {
		parts = append(parts, gchalk.WithBrightYellow().Sprintf(" %d↑", manager.TotalOutgoing()))
	}
	return strings.Join(parts, "")
}

// readUserInput reads and sanitizes user input
func readUserInput() string {
	r := bufio.NewReader(os.Stdin)
	answer, err := r.ReadString('\n')
	if err != nil {
		fmt.Fprintf(os.Stderr, "Couldn't get prompt answer: '%s'.\n", err)
		os.Exit(1)
	}
	return strings.TrimSpace(strings.ToLower(answer))
}

// processUserAnswer validates and returns the user's choice
func processUserAnswer(answer string, w *ui.StdoutWriter) string {
	if len(answer) > 1 {
		fmt.Fprintf(w, "Invalid option: '%s'\n\n", answer)
		return "invalid"
	}
	return answer
}

// showHelpOptions displays available sync options
func showHelpOptions(w *ui.StdoutWriter) {
	w.ResetToMarker()
	fmt.Fprintln(w, gchalk.Bold("Available options:"))
	fmt.Fprintf(w, "  %s = %s", gchalk.Bold("y"), "Full Sync (stash, pull+rebase, push) [default]\n")
	fmt.Fprintf(w, "  %s = %s", gchalk.Bold("n"), "No sync at all\n")
	fmt.Fprintf(w, "  %s = %s", gchalk.Bold("i"), "Sync incoming only (stash, pull+rebase)\n")
	fmt.Fprintf(w, "  %s = %s", gchalk.Bold("?"), "Explain options\n")
	fmt.Fprintln(w)
}

// updateRepositories fetches latest changes for all repositories, returning true if any failed
func updateRepositories(manager repo.RepositoryManager, renderer ui.UIRenderer) bool {
	ctx := context.Background()
	err := manager.UpdateAll(ctx, renderer.RenderProgress(false))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Some repositories failed to update: %v\n", err)
	}
	if verboseArg {
		printVerboseErrors(manager)
	}
	return err != nil
}

// syncRepositories performs sync operations on all repositories, returning true if any failed
func syncRepositories(manager repo.RepositoryManager, incomingOnly bool, renderer ui.UIRenderer) bool {
	ctx := context.Background()
	err := manager.SyncAll(ctx, incomingOnly, renderer.RenderProgress(incomingOnly))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Some repositories failed to sync: %v\n", err)
	}
	if verboseArg {
		printVerboseErrors(manager)
	}
	return err != nil
}

// printVerboseErrors prints the full git stderr for any failed repositories
func printVerboseErrors(manager repo.RepositoryManager) {
	for _, r := range manager.GetRepositories() {
		if r.Error == nil {
			continue
		}
		var gitErr *git.GitError
		if errors.As(r.Error, &gitErr) && gitErr.StdErr != "" {
			fmt.Fprintf(os.Stderr, "\n[%s] %s\n", r.Name, gitErr.StdErr)
		}
	}
}
