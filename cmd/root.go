package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"sync"

	"github.com/benweidig/tortuga/git"
	"github.com/benweidig/tortuga/repo"
	"github.com/benweidig/tortuga/ui"
	"github.com/benweidig/tortuga/version"

	"github.com/fatih/color"
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
	RootCmd.Flags().BoolVarP(&yesArg, "yes", "y", false, "Anwser 'Yes' to 'Stash/Pull/Rebase/Push' prompt")
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
		color.NoColor = true
	}

	// /////////////////////////////////////////////////////////////////////////
	// Step 2: Find repositories
	// /////////////////////////////////////////////////////////////////////////

	repos, err := findRepositories(basePath)

	bailOnErrors(err, repos)

	if len(repos) == 0 {
		fmt.Fprintf(os.Stderr, "No repositories found at '%s'.\n", basePath)
		os.Exit(1)
	}

	// /////////////////////////////////////////////////////////////////////////
	// Step 3: Update repositories
	// /////////////////////////////////////////////////////////////////////////

	updateRepositories(repos)

	bailOnErrors(nil, repos)

	// /////////////////////////////////////////////////////////////////////////
	// Step 4: Check if we can sync at all
	// /////////////////////////////////////////////////////////////////////////

	atLeastOneSyncNeeded := false
	for _, r := range repos {
		if r.NeedsSync() {
			atLeastOneSyncNeeded = true
			break
		}
	}
	if atLeastOneSyncNeeded == false {
		os.Exit(0)
	}

	// /////////////////////////////////////////////////////////////////////////
	// Step 5a: Ask if you should sync
	// /////////////////////////////////////////////////////////////////////////

	if yesArg == false {
		answer, err := ui.PromptYesNo("Stash/Rebase/Push?")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Couldn't get prompt answer: '%s'.\n", err)
			os.Exit(1)
		}

		if answer == false {
			os.Exit(0)
		}
	}

	fmt.Println()

	// /////////////////////////////////////////////////////////////////////////
	// Step 5b: Do the actual sync
	// /////////////////////////////////////////////////////////////////////////

	syncedRepos := syncRepositories(repos)

	bailOnErrors(nil, syncedRepos)

	fmt.Println()
}

func findRepositories(basePath string) ([]*repo.Repository, error) {
	var repos []*repo.Repository

	entries, err := ioutil.ReadDir(basePath)
	if err != nil {
		return repos, err
	}

	for _, entry := range entries {
		// We are only interested in directories
		if entry.IsDir() == false {
			continue
		}

		// Build paths and check if we got .git directory
		entryPath := path.Join(basePath, entry.Name())
		if git.IsPossiblyRepo(entryPath) == false {
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
	ui.WriteRepositoryStatus(w, repos)
	w.Flush()

	// 3. Start a waitgroup
	var wg sync.WaitGroup
	wg.Add(len(repos))

	// 4. Iterate over the repos and parallel check/update the repos and update the output
	for idx := range repos {
		r := repos[idx]
		go func() {
			r.Update()
			ui.WriteRepositoryStatus(w, repos)
			w.Flush()
			wg.Done()
		}()
	}

	// 5. Wait for all goroutines to finish
	wg.Wait()
}

func syncRepositories(repos []*repo.Repository) []*repo.Repository {
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

	ui.WriteRepositoryStatus(w, syncable)
	w.Flush()

	// 3. Do the work async for better speed
	if len(syncable) > 0 {
		var wg sync.WaitGroup
		wg.Add(len(syncable))

		for idx := range syncable {
			r := syncable[idx]

			go func() {
				r.Sync()
				ui.WriteRepositoryStatus(w, syncable)
				w.Flush()
				wg.Done()
			}()
		}
		wg.Wait()
	}

	return syncable
}

func bailOnErrors(err error, repos []*repo.Repository) {
	var shouldExit bool

	if err != nil {
		fmt.Fprintln(os.Stderr, "An error occured:", err)
		shouldExit = true
	}

	for _, r := range repos {
		if r.State != repo.StateError {
			continue
		}
		fmt.Fprintln(os.Stderr)
		fmt.Fprintf(os.Stderr, "%s/%s:\n", r.Name, r.Branch)
		ge, ok := r.Error.(*git.ExternalError)
		if ok {
			fmt.Fprintln(os.Stderr, ge.Cause.Error())
			if verboseArg {
				fmt.Fprintln(os.Stderr, "Git stdout:", ge.StdOut)
				fmt.Fprintln(os.Stderr, "Git stderr:", ge.StdErr)
			}
		} else {
			fmt.Fprintln(os.Stderr, r.Error.Error())
		}

		shouldExit = true
	}

	if shouldExit {
		fmt.Fprintln(os.Stderr)
		os.Exit(1)
	}
}
