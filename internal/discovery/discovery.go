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
//
// Unless noIgnores is true, any repository containing a ".tortugaignore" file
// is excluded. For strategies 1 and 3 this means ErrNoRepos is returned. For
// strategy 2 only the ignored children are dropped; if all children are
// ignored ErrNoRepos is returned rather than falling through to strategy 3.
func FindRepos(root string, noIgnores bool) ([]model.Repo, error) {
	if isGitRepo(root) {
		if !noIgnores && isIgnored(root) {
			return nil, ErrNoRepos
		}
		return reposFrom([]string{root}), nil
	}

	if children, err := gitChildren(root); err == nil && len(children) > 0 {
		if !noIgnores {
			children = filterIgnored(children)
		}
		if len(children) > 0 {
			return reposFrom(children), nil
		}
		return nil, ErrNoRepos
	}

	if ancestor, err := findAncestorRepo(root); err == nil {
		if !noIgnores && isIgnored(ancestor) {
			return nil, ErrNoRepos
		}
		return reposFrom([]string{ancestor}), nil
	}

	return nil, ErrNoRepos
}

// isIgnored reports whether path contains a ".tortugaignore" file.
func isIgnored(path string) bool {
	_, err := os.Stat(filepath.Join(path, ".tortugaignore"))
	return err == nil
}

// filterIgnored returns a new slice with any paths containing ".tortugaignore" removed.
func filterIgnored(paths []string) []string {
	result := make([]string, 0, len(paths))
	for _, p := range paths {
		if !isIgnored(p) {
			result = append(result, p)
		}
	}
	return result
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
