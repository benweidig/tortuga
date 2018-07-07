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
	localOnlyArg  bool
	monochromeArg bool
	yesArg        bool
)

// RootCmd is the only command, so this is Tortuga
var RootCmd = &cobra.Command{
	Version: version.BuildVersion(),
	Use:     "tt",
	Short:   "Tortuga",
	Args:    cobra.MaximumNArgs(1),
	Long:    "CLI tool for fetching/pushing/rebasing multiple git repositories at once",
	Run:     runCommand,
}

func init() {
	RootCmd.Flags().BoolVarP(&localOnlyArg, "local-only", "l", false, "Local mode, don't fetch remotes")
	RootCmd.Flags().BoolVarP(&monochromeArg, "monochrome", "m", false, "Monochrome output, no ANSI colorize")
	RootCmd.Flags().BoolVarP(&yesArg, "yes", "y", false, "Anwser 'Yes' to 'Stash/Pull/Rebase/Push' prompt")
}

func runCommand(_ *cobra.Command, args []string) {

	// Step 1: Parse arguments and prepare requirements

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

	// Step 2: Find and Update the repositoriers

	repos := findAndUpdate(basePath)

	// If no remote actions are supposed to be done we need to end here
	if localOnlyArg {
		os.Exit(0)
	}

	// Check if we actual can do any work at all
	atLeastOneDirty := false
	for _, r := range repos {
		if r.IsDirty() {
			atLeastOneDirty = true
			break
		}
	}
	if atLeastOneDirty == false {
		os.Exit(0)
	}

	// Step 3: Ask if we should up

	// There's is work to do, ask if we should
	if yesArg == false {
		answer, err := ui.PromptYesNo("Stash/Pull/Rebase/Push?")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Couldn't get prompt answer: '%s'.\n", err)
			os.Exit(1)
		}
		if answer == false {
			os.Exit(0)
		}
	}

	fmt.Println()

	syncRepositories(repos)

	fmt.Println()
}

func findAndUpdate(basePath string) []*repo.Repository {
	// 1. Find all available repositories
	repos, err := findRepos(basePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "An error occured: '%s'.\n", err)
		os.Exit(1)
	}

	// 2. We need at least one repository
	if len(repos) == 0 {
		fmt.Fprintf(os.Stderr, "No repositories found at '%s'.\n", basePath)
		os.Exit(1)
	}

	// 3. Start live writer and render the repositories
	w := ui.NewStdoutWriter()

	// 4. Initial output showing all repos
	writeCurrentStatus(w, repos)
	w.Flush()

	// 5. Start a waitgroup
	var wg sync.WaitGroup
	wg.Add(len(repos))

	// 6. Iterate over the repos and parallel check/update the repos and update the output
	for idx := range repos {
		r := repos[idx]
		go func() {
			r.UpdateChanges(localOnlyArg)
			writeCurrentStatus(w, repos)
			w.Flush()
			wg.Done()
		}()
	}

	// 7. Wait for all goroutines to finish
	wg.Wait()

	return repos
}

func syncRepositories(repos []*repo.Repository) {
	// 1. Separate repos by safely doable and not so safe
	var syncable, safe, unsafe []*repo.Repository

	for idx := range repos {
		r := repos[idx]

		// No need to check an unsafe repository
		if r.State == repo.StateError {
			continue
		}

		// The refs are different, so we need to do some work.
		if r.Incoming > 0 || r.Outgoing > 0 {
			// We separate the repos to 2 categories: safe and unsafe.
			//
			// Possible states for a repository to be considered safe:
			// - no outgoing changesets and nothing to stash -> Only incoming changes
			// - Outgoing changes without incoming
			//
			// Unsafe means there's a possibility for conflicts due to a merge/rebase or stashing/unstashing
			// the current work tree state
			if (r.Outgoing == 0 && r.LocalChanges.Stashable == 0) || (r.Incoming == 0 && r.Outgoing > 0) {
				safe = append(safe, r)
			} else {
				unsafe = append(unsafe, r)
			}
			syncable = append(syncable, r)

		} else {
			r.State = repo.StateSynced
		}
	}

	// 2. Start live writer and render the repositories
	w := ui.NewStdoutWriter()

	writeActionsTaken(w, syncable)
	w.Flush()

	// 3. Do work for safe repositories first
	var wg sync.WaitGroup
	if len(safe) > 0 {
		wg.Add(len(safe))

		for idx := range safe {
			r := safe[idx]

			go func() {
				err := r.Sync()
				if err != nil {
					// Ignore error, it will be displayed
				}

				writeActionsTaken(w, syncable)
				w.Flush()
				wg.Done()
			}()
		}
		wg.Wait()
	}

	// 4. Do the unsafe repositories in sync fashion
	for idx := range unsafe {
		r := unsafe[idx]

		err := r.Sync()
		if err != nil {
			// Ignore error, it will be displayed
		}

		writeActionsTaken(w, syncable)
		w.Flush()
	}
}

func findRepos(basePath string) ([]*repo.Repository, error) {
	var repos []*repo.Repository

	entries, err := ioutil.ReadDir(basePath)
	if err != nil {
		return repos, nil
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
