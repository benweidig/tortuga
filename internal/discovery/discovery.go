package discovery

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/benweidig/tortuga/internal/git"
	"github.com/benweidig/tortuga/internal/model"
)

var ErrNoRepos = errors.New("no git repositories found")

// FindRepos discovers git repositories relative to root using three strategies
// in order, stopping at the first that produces results:
//
//  1. Root is itself a git repo
//  2. Direct children of root are git repos
//  3. Walk upward from root to find a parent repo
func FindRepos(root string) ([]model.Repo, error) {
	if isGitRepo(root) {
		return reposFrom([]string{root}), nil
	}

	if children, err := gitChildren(root); err == nil && len(children) > 0 {
		return reposFrom(children), nil
	}

	if ancestor, err := findAncestorRepo(root); err == nil {
		return reposFrom([]string{ancestor}), nil
	}

	return nil, ErrNoRepos
}

// isGitRepo reports whether path contains a .git entry (file or directory).
func isGitRepo(path string) bool {
	_, err := os.Stat(filepath.Join(path, ".git"))
	return err == nil
}

// gitChildren returns the paths of direct children of root that are git repos,
// in the order returned by os.ReadDir (alphabetical).
func gitChildren(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	var paths []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}

		child := filepath.Join(root, e.Name())

		if isGitRepo(child) {
			paths = append(paths, child)
		}
	}

	return paths, nil
}

// findAncestorRepo walks upward from path until it finds a directory containing
// .git, stopping at the filesystem root.
func findAncestorRepo(path string) (string, error) {
	for {
		parent := filepath.Dir(path)

		if parent == path {
			// Reached filesystem root without finding a repo
			return "", ErrNoRepos
		}
		path = parent

		if isGitRepo(path) {
			return path, nil
		}
	}
}

// reposFrom builds a []model.Repo from a list of paths, detecting the branch
// for each.
func reposFrom(paths []string) []model.Repo {
	repos := make([]model.Repo, 0, len(paths))

	for _, p := range paths {
		branch, err := git.LocalBranch(p)

		if err != nil {
			// Detached HEAD — LocalBranch returns an error and the SHA/descriptor
			branch = "(detached)"
		}

		repos = append(repos, model.Repo{
			Path:   p,
			Name:   filepath.Base(p),
			Branch: branch,
		})
	}

	return repos
}
