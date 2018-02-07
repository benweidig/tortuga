package repo

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
)

type Repository struct {
	path string

	Name     string
	Branch   string
	Changes  Changes
	Incoming int
	Outgoing int
	State    State
}

func NewRepository(repoPath string) (Repository, error) {
	r := Repository{
		Name:  path.Base(repoPath),
		path:  repoPath,
		State: StateNone,
	}

	branch, err := r.currentBranch()
	if err != nil {
		fmt.Println(err)
		log.Fatal(fmt.Sprintf("Couldn't determinate branch. ", err.Error()))
	}

	r.Branch = branch

	return r, nil
}

// Updates a Repository with the current changes and number of incoming/outgoing.
// If localOnly is true no fetching fo the remote will occur.
func (r *Repository) Update(localOnly bool) error {
	if localOnly == false {
		_, _, err := r.git("fetch", "--all")
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

func (r *Repository) Sync() error {
	if r.Changes.Stashable > 0 {
		_, _, err := r.git("stash")
		if err != nil {
			r.State = StateError
			return err
		}
	}

	if r.Incoming > 0 {
		_, _, err := r.git("pull", "--rebase")
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
		count += 1
	}

	err = scanner.Err()
	if err != nil {
		return 0, err
	}

	return count, nil
}
