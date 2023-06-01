package cmd

import (
	"bufio"
	"fmt"
	"io/ioutil"
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
	RootCmd.Flags().BoolVarP(&verboseArg, "verbose", "v", false, "Verbose error output")
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

	updateRepositories(repos)

	printErrors(repos)

	// /////////////////////////////////////////////////////////////////////////
	// Step 4: Check if we can sync at all
	// /////////////////////////////////////////////////////////////////////////

	incoming := 0
	outgoing := 0

	var syncableRepos []*repo.Repository

	for _, r := range repos {
		incoming += r.Incoming
		outgoing += r.Outgoing
		if r.NeedsSync() {
			syncableRepos = append(syncableRepos, r)
		}
	}
	if incoming == 0 && outgoing == 0 {
		os.Exit(0)
	}

	// /////////////////////////////////////////////////////////////////////////
	// Step 5a: Ask if you should sync
	// /////////////////////////////////////////////////////////////////////////

	var syncIncomingOnly bool

	if !yesArg {
		for {
			prompt := ""
			if incoming > 0 {
				prompt += gchalk.WithBrightYellow().Sprintf(" %d↓", incoming)
			}

			if outgoing > 0 {
				prompt += gchalk.WithBrightYellow().Sprintf(" %d↑", outgoing)
			}

			fmt.Printf("%s Sync Changes?%s [Y/n/i/?] ", gchalk.WithWhite().Bold(">>>"), prompt)

			r := bufio.NewReader(os.Stdin)
			answer, err := r.ReadString('\n')
			if err != nil {
				fmt.Fprintf(os.Stderr, "Couldn't get prompt answer: '%s'.\n", err)
				os.Exit(1)
			}

			// Sanitize
			answer = strings.TrimSpace(strings.ToLower(answer))
			if len(answer) > 1 {
				fmt.Fprintf(os.Stderr, "Invalid option: '%s'\n\n", answer)
				continue
			}

			if answer == "n" {
				os.Exit(0)
			} else if answer == "i" {
				syncIncomingOnly = true
				break
			} else if answer == "?" {
				fmt.Println()
				fmt.Println(gchalk.Bold("Available options:"))
				fmt.Printf("  %s = %s", gchalk.Bold("y"), "Full Sync (stash, pull+rebase, push) [default]\n")
				fmt.Printf("  %s = %s", gchalk.Bold("n"), "No sync at all\n")
				fmt.Printf("  %s = %s", gchalk.Bold("i"), "Sync incoming only (stash, pull+rebase)\n")
				fmt.Printf("  %s = %s", gchalk.Bold("?"), "Explain options\n")
				fmt.Println()
			} else if answer == "y" || answer == "" {
				break
			}
		}
	}

	fmt.Println()

	// /////////////////////////////////////////////////////////////////////////
	// Step 5b: Do the actual sync
	// /////////////////////////////////////////////////////////////////////////

	syncedRepos := syncRepositories(syncableRepos, syncIncomingOnly)

	printErrors(syncedRepos)

	fmt.Println()
}

func findRepositories(basePath string) ([]*repo.Repository, error) {
	var repos []*repo.Repository

	if git.IsPossiblyRepo(basePath) {
		r, err := repo.NewRepository(basePath)
		repos = append(repos, r)
		return repos, err
	}

	entries, err := ioutil.ReadDir(basePath)
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
		if !git.IsPossiblyRepo(entryPath) {
			continue
		}

		// Build repository. We ignore errors so all will be displayed
		r, _ := repo.NewRepository(entryPath)
		repos = append(repos, r)
	}

	return repos, nil
}

func updateRepositories(repos []*repo.Repository) {
	// 1. Start live writer and render the repositories
	w := ui.NewStdoutWriter()

	// 2. Initial output showing all repos
	ui.WriteRepositoryStatus(w, repos, false)
	w.Flush()

	// 3. Start a waitgroup
	var wg sync.WaitGroup
	wg.Add(len(repos))

	// 4. Iterate over the repos and parallel check/update the repos and update the output
	for idx := range repos {
		r := repos[idx]
		go func() {
			r.Update()
			ui.WriteRepositoryStatus(w, repos, false)
			w.Flush()
			wg.Done()
		}()
	}

	// 5. Wait for all goroutines to finish
	wg.Wait()
}

func syncRepositories(repos []*repo.Repository, incomingOnly bool) []*repo.Repository {
	// 1. Find the repos that are needed to by synced
	var syncable []*repo.Repository

	for idx := range repos {
		r := repos[idx]

		// No need to check an unsafe repository
		if r.State == repo.StateError {
			continue
		}

		if r.NeedsSync() {
			r.State = repo.StateNeedsSync
			syncable = append(syncable, r)
		} else {
			r.State = repo.StateNoSyncNeeded
		}
	}

	// 2. Start live writer and render the repositories
	w := ui.NewStdoutWriter()

	ui.WriteRepositoryStatus(w, syncable, incomingOnly)
	w.Flush()

	// 3. Do the work async for better speed
	if len(syncable) > 0 {
		var wg sync.WaitGroup
		wg.Add(len(syncable))

		for idx := range syncable {
			r := syncable[idx]

			go func() {
				r.Sync(incomingOnly)
				ui.WriteRepositoryStatus(w, syncable, incomingOnly)
				w.Flush()
				wg.Done()
			}()
		}
		wg.Wait()
	}

	return syncable
}

func printErrors(repos []*repo.Repository) {
	errorCount := repo.ErrorCount(repos)
	if errorCount == 0 {
		return
	}

	fmt.Fprintln(os.Stderr, gchalk.WithRed().Sprintf("Errors occured: %d\n", errorCount))

	if !verboseArg {
		return
	}

	for _, r := range repos {
		if r.State != repo.StateError {
			continue
		}
		fmt.Fprintln(os.Stderr)
		fmt.Fprintln(os.Stderr, gchalk.WithRed().Sprintf("%s/%s:", r.Name, r.Branch))
		ge, ok := r.Error.(*git.ExternalError)
		if ok {
			fmt.Fprintln(os.Stderr, ge.StdErr)
		} else {
			fmt.Fprintln(os.Stderr, r.Error.Error())
		}
	}
}
