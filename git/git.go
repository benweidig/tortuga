package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"
)

// IsAvailable checks if git executable is available in the path
func IsAvailable() error {
	_, err := exec.LookPath("git")
	return err
}

// IsPossiblyRepo tries to determinate if the a path is repo by checking for a '.git' folder
func IsPossiblyRepo(basePath string) bool {
	gitPath := path.Join(basePath, ".git")
	stat, err := os.Stat(gitPath)
	if err != nil {
		return false
	}
	if stat.IsDir() == false {
		return false
	}

	return true
}

func git(path string, args ...string) (bytes.Buffer, error) {
	// Disable terminal prompting so it fails if credentials are needed etc.
	err := os.Setenv("GIT_TERMINAL_PROMPT", "0")
	if err != nil {
		return bytes.Buffer{}, err
	}

	// Combine args and build command
	args = append([]string{"-C", path}, args...)
	cmd := exec.Command("git", args...)

	// Attach buffers, a function might need both so just grab'em
	var outBuffer bytes.Buffer
	var errBuffer bytes.Buffer
	cmd.Stdout = &outBuffer
	cmd.Stderr = &errBuffer

	// Run command, but don't handle errors here, this is just a helper function
	err = cmd.Run()

	if err != nil {
		err = wrapError(err, outBuffer, errBuffer)
	}

	return outBuffer, err
}

// LocalBranch returns the local branch name of the current HEAD
func LocalBranch(path string) (string, error) {
	stdOut, err := git(path, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}

	// We have to sanitize the output for easier usage
	branch := strings.TrimSpace(stdOut.String())

	return branch, nil
}

// UpstreamBranch returns the name of the upstream branch
func UpstreamBranch(path string) (string, error) {
	stdOut, err := git(path, "rev-parse", "--symbolic-full-name", "--abbrev-ref", "@{u}")
	if err != nil {
		return "", err
	}

	// We have to sanitize the output for easier usage
	branch := strings.TrimSpace(stdOut.String())

	return branch, nil
}

func commitsCount(path string, rangeSpecifier string) (int, error) {
	stdOut, err := git(path, "rev-list", rangeSpecifier)
	if err != nil {
		return -1, err
	}

	count := bytes.Count(stdOut.Bytes(), []byte("\n"))

	return count, nil
}

// Incoming counts the incoming commits (head vs upstream)
func Incoming(path string, branch string) (int, error) {
	return commitsCount(path, fmt.Sprintf("HEAD..%s@{upstream}", branch))
}

// Outgoing counts the outgoing commits (push vs head)
func Outgoing(path string, branch string) (int, error) {
	return commitsCount(path, fmt.Sprintf("%s@{push}..HEAD", branch))
}

// Fetch fetches the specified remote
func Fetch(path string, remote string) error {
	_, err := git(path, "fetch", remote)
	return err
}

// Status returns a parseable (--porcelain) status
func Status(path string) (bytes.Buffer, error) {
	status, err := git(path, "status", "--porcelain")
	return status, err
}

// Rebase tries to rebase the current working tree with the upstream
func Rebase(path string) error {
	_, err := git(path, "rebase", "@{u}")
	return err
}

// Push pushes the repository to the remote
func Push(path string) error {
	_, err := git(path, "push")
	return err
}

// Stash stashes the current working tree
func Stash(path string) error {
	_, err := git(path, "stash", "save")

	return err
}

// StashPop pops the last stash
func StashPop(path string) error {
	_, err := git(path, "stash", "pop")
	return err
}
