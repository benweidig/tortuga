package cmd

import (
	"bufio"
	"errors"
	"io/ioutil"
	"log"
	"os"
	"path"
	"sync"

	"fmt"
	"strings"

	"github.com/benweidig/cli-table"
	"github.com/benweidig/tortuga/repo"
	"github.com/fatih/color"
	"github.com/gosuri/uilive"
	"github.com/spf13/cobra"
)

var (
	localOnlyArg bool
)

var RootCmd = &cobra.Command{
	Version: "0.0.1",
	Use:     "tt",
	Short:   "Tortuga",
	Args:    cobra.MaximumNArgs(1),
	Long:    "CLI tool for fetching/pushing/rebasing multiple git repositories at once",
	Run:     runCommand,
}

func init() {
	RootCmd.Flags().BoolVarP(&localOnlyArg, "local-only", "l", false, "Local mode, don't fetch remotes")
}

func runCommand(_ *cobra.Command, args []string) {
	var basePath string

	if len(args) == 1 {
		basePath = args[0]
	} else {
		wd, err := os.Getwd()
		if err != nil {
			log.Fatal("Couldn't retrieve working directory. " + err.Error())
		}
		basePath = wd
	}

	repos := findAndUpdate(basePath)

	if localOnlyArg {
		os.Exit(0)
	}

	answer, err := askYN("Stash/Pull/Rebase/Push?")
	if err != nil {
		log.Fatal(err)
	}
	if answer == false {
		os.Exit(0)
	}

	fmt.Println()

	syncRepositories(repos)

	fmt.Println()
}

func findAndUpdate(basePath string) []repo.Repository {
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
	renderRepoTable(w, repos)

	// 5. Start a waitgroup
	var wg sync.WaitGroup
	wg.Add(len(repos))

	// 6. Iterate over the repos and parallel check/update the repos and update the output
	for idx := range repos {
		r := &repos[idx]
		go func() {
			defer wg.Done()
			r.Update(localOnlyArg)
			renderRepoTable(w, repos)
		}()
	}

	// 7. Wait for all goroutines to finish
	wg.Wait()

	// 8. We're done, stop live-writer
	w.Stop()

	return repos
}

func syncRepositories(repos []repo.Repository) {
	// 1. Separate repos by safely doable and not so safe
	var safeRepos []*repo.Repository
	var unsafeRepos []*repo.Repository

	for idx := range repos {
		r := &repos[idx]

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
			if (r.Outgoing == 0 && r.Changes.Stashable == 0) || (r.Incoming == 0 && r.Outgoing > 0) {
				safeRepos = append(safeRepos, r)
			} else {
				unsafeRepos = append(unsafeRepos, r)
			}
		} else {
			r.State = repo.StateSynced
		}
	}

	// 2. Start live writer and render the repositories
	w := uilive.New()
	w.Start()

	renderSyncTable(w, repos)

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
					// TODO: Error Handling
				}

				renderSyncTable(w, repos)
			}()
		}
		wg.Wait()
	}

	// 4. Do the unsafe repositories in sync fashion
	for idx := range unsafeRepos {
		r := unsafeRepos[idx]

		err := r.Sync()
		if err != nil {
			log.Fatal(err)
		}
		log.Fatal(err)

		r.State = repo.StateSynced
		renderSyncTable(w, repos)
	}

	// 5. We're done, stop live-writer
	w.Stop()
}

func findRepos(basePath string) ([]repo.Repository, error) {
	var repos []repo.Repository

	entries, err := ioutil.ReadDir(basePath)
	if err != nil {
		return repos, nil
	}

	for _, entry := range entries {
		// We are only interested in directories
		if entry.IsDir() == false {
			continue
		}

		entryPath := path.Join(basePath, entry.Name())
		gitPath := path.Join(entryPath, ".git")
		stat, err := os.Stat(gitPath)
		if err != nil {
			continue
		}

		if stat.IsDir() == false {
			continue
		}

		r, err := repo.NewRepository(entryPath)

		if err != nil {
			return repos, err
		}
		repos = append(repos, r)
	}

	return repos, nil
}

func renderRepoTable(w *uilive.Writer, repos []repo.Repository) {
	table := clitable.New()
	table.AddRow("PROJECT", "BRANCH", "STATUS")

	for _, r := range repos {
		var status string
		if r.State == repo.StateUpdated {
			var statusParts []string
			if r.Changes.Total == 0 {
				statusParts = append(statusParts, color.GreenString("%d*", r.Changes.Total))
			} else {
				statusParts = append(statusParts, color.YellowString("%d*", r.Changes.Total))
			}

			if r.Incoming > 0 {
				statusParts = append(statusParts, color.YellowString("%d↓", r.Incoming))
			}
			if r.Outgoing > 0 {
				statusParts = append(statusParts, color.YellowString("%d↑", r.Outgoing))
			}
			status = strings.Join(statusParts, " ")
		} else {
			status = "..."
		}
		table.AddRow(color.WhiteString(r.Name), color.WhiteString(r.Branch), status)
	}

	fmt.Fprintln(w, table)

	w.Flush()
}

func renderSyncTable(w *uilive.Writer, repos []repo.Repository) {
	table := clitable.New()
	table.AddRow("PROJECT", "BRANCH", "STATUS")

	for _, r := range repos {
		var status string
		if r.State == repo.StateSynced {
			if r.Outgoing == 0 && r.Incoming == 0 {
				status = "Nothing to do"
			} else {
				var statusParts []string
				if r.Incoming > 0 {
					statusParts = append(statusParts, fmt.Sprintf("%d pulled", r.Incoming))
				}
				if r.Outgoing > 0 {
					statusParts = append(statusParts, fmt.Sprintf("%d pushed", r.Outgoing))
				}
				status = strings.Join(statusParts, ", ")
			}
			status = color.GreenString(status)
		} else {
			status = "..."
		}
		table.AddRow(color.New(color.FgWhite).Sprint(r.Name), color.New(color.FgWhite).Sprint(r.Branch), color.New(color.FgWhite).Sprint(status))
	}

	fmt.Fprintln(w, table)

	w.Flush()
}

func askYN(question string) (bool, error) {
	color.New(color.Bold).Print(">>> ")
	fmt.Printf("%s [Y/n] ", question)

	r := bufio.NewReader(os.Stdin)
	answer, err := r.ReadString('\n')
	if err != nil {
		return false, err
	}

	if len(answer) > 2 {
		msg := fmt.Sprintf("Invalid option: %s", answer)
		return false, errors.New(msg)
	}

	if strings.ToLower(answer) == "n\n" {
		return false, nil
	}

	return true, nil

}
