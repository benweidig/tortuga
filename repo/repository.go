package repo

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
)

// Repository represents Git repository, but only the currently checked out branch
type Repository struct {
	path string

	Name     string
	Branch   string
	Changes  Changes
	Incoming int
	Outgoing int
	State    State
}

// NewRepository creates a bare Repository construct containing the minimum for initial display
func NewRepository(repoPath string) (Repository, error) {
	r := Repository{
		Name:  path.Base(repoPath),
		path:  repoPath,
		State: StateNone,
	}

	branch, err := r.currentBranch()
	if err != nil {
		r.State = StateError
		r.Branch = "???"
		return r, nil
	}

	r.Branch = branch

	return r, nil
}

// Update gets and sets the current changes and number of incoming/outgoing of a Repository.
// If localOnly is true no fetching fo the remote will occur.
func (r *Repository) Update(localOnly bool) error {
	if r.State == StateError {
		return nil
	}

	if localOnly == false {
		_, _, err := r.git("fetch", "origin")
		if err != nil {
			r.State = StateError
			return err
		}
	}

	status, _, err := r.git("status", "--porcelain")
	if err != nil {
		r.State = StateError
		return err
	}

	r.Changes = NewChanges(status)

	incoming, err := r.commitDiff(fmt.Sprintf("HEAD..%s@{upstream}", r.Branch))
	if err != nil {
		r.State = StateError
		return err
	}
	r.Incoming = incoming

	outgoing, err := r.commitDiff(fmt.Sprintf("%s@{push}..HEAD", r.Branch))
	if err != nil {
		r.State = StateError
		return err
	}
	r.Outgoing = outgoing

	r.State = StateUpdated

	return nil
}

// Sync stashes, rebases, pushs and unstashes the Repository
func (r *Repository) Sync() error {
	if r.State == StateError {
		return nil
	}

	if r.Changes.Stashable > 0 {
		_, _, err := r.git("stash")
		if err != nil {
			r.State = StateError
			return err
		}
	}

	if r.Incoming > 0 {
		_, _, err := r.git("rebase", "@{u}")
		if err != nil {
			r.State = StateError
			return err
		}
	}

	if r.Outgoing > 0 {
		_, _, err := r.git("push")
		if err != nil {
			r.State = StateError
			return err
		}
	}

	if r.Changes.Stashable > 0 {
		_, _, err := r.git("stash", "pop")
		if err != nil {
			r.State = StateError
			return err
		}
	}

	r.State = StateSynced

	return nil
}

// Runs a git command with the specified args against a path
func (r Repository) git(args ...string) (bytes.Buffer, bytes.Buffer, error) {
	// Disable terminal prompting so it fails if credentials are needed
	os.Setenv("GIT_TERMINAL_PROMPT", "0")

	// Combine args and build command
	args = append([]string{"-C", r.path}, args...)
	cmd := exec.Command("git", args...)

	// Attach buffers, a function might need both so just grab both
	var outBuffer bytes.Buffer
	var errBuffer bytes.Buffer
	cmd.Stdout = &outBuffer
	cmd.Stderr = &errBuffer

	// Run command, but don't handle errors here, this is just a helper function
	err := cmd.Run()

	return outBuffer, errBuffer, err
}

// Returns the current branch of the repository
func (r Repository) currentBranch() (string, error) {
	stdOut, _, err := r.git("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}

	// We have to sanitize the output for easier usage
	branch := stdOut.String()
	branch = strings.TrimSuffix(branch, "\n")

	return branch, nil
}

func (r Repository) commitDiff(rangeSpecifier string) (int, error) {
	stdOut, _, err := r.git("rev-list", rangeSpecifier)
	if err != nil {
		return 0, err
	}

	scanner := bufio.NewScanner(&stdOut)
	count := 0
	for scanner.Scan() {
		count++
	}

	err = scanner.Err()
	if err != nil {
		return 0, err
	}

	return count, nil
}
