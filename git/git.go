package git

import (
	"bufio"
	"bytes"
	"os"
	"os/exec"
	"strings"
)

// Run runs a git command with the specified args against a path
func Run(path string, args ...string) (bytes.Buffer, bytes.Buffer, error) {
	// Disable terminal prompting so it fails if credentials are needed etc.
	os.Setenv("GIT_TERMINAL_PROMPT", "0")

	// Combine args and build command
	args = append([]string{"-C", path}, args...)
	cmd := exec.Command("git", args...)

	// Attach buffers, a function might need both so just grab both
	var outBuffer bytes.Buffer
	var errBuffer bytes.Buffer
	cmd.Stdout = &outBuffer
	cmd.Stderr = &errBuffer

	// Run command, but don't handle errors here, this is just a helper function
	err := cmd.Run()

	// Try to  concretize error for better usage later on
	err = ConcretizeError(err, errBuffer)

	return outBuffer, errBuffer, err
}

// LocalBranch returns the local branch name of the current HEAD
func LocalBranch(path string) (string, error) {
	stdOut, _, err := Run(path, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}

	// We have to sanitize the output for easier usage
	branch := stdOut.String()
	branch = strings.TrimSuffix(branch, "\n")

	return branch, nil
}

// UpstreamBranch returns the name of the upstream branch
func UpstreamBranch(path string) (string, error) {
	stdOut, _, err := Run(path, "rev-parse", "--symbolic-full-name", "--abbrev-ref", "@{u}")
	if err != nil {
		return "", err
	}

	// We have to sanitize the output for easier usage
	branch := stdOut.String()
	branch = strings.TrimSuffix(branch, "\n")

	return branch, nil
}

// CommitsCount returns the number of commits in the range
func CommitsCount(path string, rangeSpecifier string) (int, error) {
	stdOut, _, err := Run(path, "rev-list", rangeSpecifier)
	if err != nil {
		return -1, err
	}

	scanner := bufio.NewScanner(&stdOut)
	count := 0
	for scanner.Scan() {
		count++
	}

	err = scanner.Err()
	if err != nil {
		return -1, err
	}

	return count, nil
}

// Fetch fetches the specified remote
func Fetch(path string, remote string) error {
	_, _, err := Run(path, "fetch", remote)
	return err
}

// Status returns a parseable (--porcelain) status
func Status(path string) (bytes.Buffer, error) {
	status, _, err := Run(path, "status", "--porcelain")
	return status, err
}

// Rebase tries to rebase the current working tree with the upstream
func Rebase(path string) error {
	_, _, err := Run(path, "rebase", "@{u}")
	return err
}

// Push pushes the repository to the remote
func Push(path string) error {
	_, _, err := Run(path, "push")
	return err
}

// Stash stashes the current working tree
func Stash(path string) error {
	_, _, err := Run(path, "stash")
	return err
}

// StashPop pops the last stash
func StashPop(path string) error {
	_, _, err := Run(path, "stash", "pop")
	return err
}
