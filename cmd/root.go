package cmd

import (
	"io/ioutil"
	"log"
	"os"
	"path"
	"sync"

	"fmt"

	"github.com/benweidig/tortuga/repo"
	"github.com/benweidig/tortuga/version"
	"github.com/fatih/color"
	"github.com/gosuri/uilive"
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
	RootCmd.Flags().BoolVarP(&yesArg, "yes", "y", false, "Prompt yes to Stash/Pull/Rebase/Push")
}

func runCommand(_ *cobra.Command, args []string) {
	// Determinate the directory to check.
	var basePath string

	// There can only be 0 or 1 arguments, so this check is enough
	if len(args) == 1 {
		basePath = args[0]
	} else {
		// Falback to actual working directory
		wd, err := os.Getwd()
		if err != nil {
			log.Fatal("Couldn't retrieve working directory. " + err.Error())
		}
		basePath = wd
	}

	// Disable colors if requested. Don't set it directly, the color library tries to check if
	// the terminal actual supports colors beforehand and disables it.
	if monochromeArg {
		color.NoColor = true
	}

	repos := findAndUpdate(basePath)

	// If no remote actions are supposed to be done we need to end here
	if localOnlyArg {
		fmt.Println("Local only, exiting...")
		os.Exit(0)
	}

	// Check if we actual can do any work at all
	atLeastOneDirty := false
	for _, r := range repos {
		if r.Incoming > 0 || r.Outgoing > 0 {
			atLeastOneDirty = true
			break
		}
	}
	if atLeastOneDirty == false {
		fmt.Println("No work can be done, exiting...")
		os.Exit(0)
	}

	// There's is work to do, ask if we should
	if yesArg == false {
		answer, err := askQuestionYN("Stash/Pull/Rebase/Push?")
		if err != nil {
			log.Fatal(err)
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
		log.Fatal(err)
	}

	// 2. We need at least one repository
	if len(repos) == 0 {
		log.Fatalf("No repositories found at '%s'", basePath)
	}

	// 3. Start live writer and render the repositories
	w := uilive.New()
	w.Start()

	// 4. Initial output showing all repos
	renderCurrentStatus(w, repos)

	// 5. Start a waitgroup
	var wg sync.WaitGroup
	wg.Add(len(repos))

	// 6. Iterate over the repos and parallel check/update the repos and update the output
	for idx := range repos {
		r := repos[idx]
		go func() {
			defer wg.Done()
			r.Update(localOnlyArg)
			renderCurrentStatus(w, repos)
		}()
	}

	// 7. Wait for all goroutines to finish
	wg.Wait()

	// 8. We're done, stop live-writer
	w.Stop()

	return repos
}

func syncRepositories(repos []*repo.Repository) {
	// 1. Separate repos by safely doable and not so safe
	var syncableRepos []*repo.Repository
	var safeRepos []*repo.Repository
	var unsafeRepos []*repo.Repository

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
				safeRepos = append(safeRepos, r)
			} else {
				unsafeRepos = append(unsafeRepos, r)
			}
			syncableRepos = append(syncableRepos, r)

		} else {
			r.State = repo.StateSynced
		}
	}

	// 2. Start live writer and render the repositories
	w := uilive.New()
	w.Start()

	renderActionsTaken(w, syncableRepos)

	// 3. Do work for safe repositories first
	var wg sync.WaitGroup
	if len(safeRepos) > 0 {
		wg.Add(len(safeRepos))

		for idx := range safeRepos {
			r := safeRepos[idx]

			go func() {
				defer wg.Done()
				err := r.Sync()
				if err != nil {
					// Ignore error, it will be displayed
				}

				renderActionsTaken(w, syncableRepos)
			}()
		}
		wg.Wait()
	}

	// 4. Do the unsafe repositories in sync fashion
	for idx := range unsafeRepos {
		r := unsafeRepos[idx]

		err := r.Sync()
		if err != nil {
			// Ignore error, it will be displayed
		}

		renderActionsTaken(w, syncableRepos)
	}

	// 5. We're done, stop live-writer
	w.Stop()
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
		gitPath := path.Join(entryPath, ".git")
		stat, err := os.Stat(gitPath)
		if err != nil {
			continue
		}
		if stat.IsDir() == false {
			continue
		}

		// Build repository. We ignore errors so all will be displayed
		r, _ := repo.NewRepository(entryPath)
		repos = append(repos, r)
	}

	return repos, nil
}
