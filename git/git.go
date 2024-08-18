package git

import (
	"bytes"
	"errors"
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

// IsRepo tries to determinate if the a path is repo by checking for a '.git' folder
func IsRepo(basePath string) bool {
	gitPath := path.Join(basePath, ".git")
	stat, err := os.Stat(gitPath)
	return err == nil && stat.IsDir()
}

func git(repoPath string, args ...string) (bytes.Buffer, error) {
	// Disable terminal prompting so it fails if credentials are needed etc.
	err := os.Setenv("GIT_TERMINAL_PROMPT", "0")
	if err != nil {
		return bytes.Buffer{}, err
	}

	// Combine args and build command
	args = append([]string{"-C", repoPath}, args...)
	cmd := exec.Command("git", args...)

	// Attach buffers, a function might need both so just grab'em
	var outBuffer bytes.Buffer
	var errBuffer bytes.Buffer
	cmd.Stdout = &outBuffer
	cmd.Stderr = &errBuffer

	// Run command, but don't handle errors here, this is just a helper function
	err = cmd.Run()

	if err != nil {
		err = wrapError(err, errBuffer)
	}
	return outBuffer, err
}

// LocalBranch returns the local branch name of the current HEAD
func LocalBranch(repoPath string) (string, error) {
	stdOut, err := git(repoPath, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}

	// We have to sanitize the output for easier usage
	branch := strings.TrimSpace(stdOut.String())

	if branch == "HEAD" {
		err = errors.New("not on a branch")
	}

	return branch, err
}

// UpstreamBranch returns the name of the upstream branch
func UpstreamBranch(repoPath string) (string, error) {
	stdOut, err := git(repoPath, "rev-parse", "--symbolic-full-name", "--abbrev-ref", "@{u}")
	if err != nil {
		return "", err
	}

	// We have to sanitize the output for easier usage
	branch := strings.TrimSpace(stdOut.String())

	return branch, nil
}

func RevList(repoPath string, rangeSpecifier string) ([]string, error) {
	stdOut, err := git(repoPath, "rev-list", rangeSpecifier)
	if err != nil {
		return []string{}, err
	}

	commits := strings.FieldsFunc(stdOut.String(), func(r rune) bool {
		return r == '\n'
	})

	return commits, nil
}

// Incoming counts the incoming commits (head vs upstream)
func Incoming(repoPath string, branch string) (int, error) {
	commits, err := RevList(repoPath, fmt.Sprintf("HEAD..%s@{upstream}", branch))
	return len(commits), err
}

// Outgoing counts the outgoing commits (push vs head)
func Outgoing(repoPath string, branch string) (int, error) {
	commits, err := RevList(repoPath, fmt.Sprintf("%s@{push}..HEAD", branch))
	return len(commits), err
}

// Fetch fetches the specified remote
func Fetch(repoPath string, remote string) error {
	_, err := git(repoPath, "fetch", remote)
	return err
}

// Status returns a parseable (--porcelain) status
func Status(repoPath string) (bytes.Buffer, error) {
	return git(repoPath, "status", "--porcelain")
}

// Rebase tries to rebase the current working tree with the upstream
func Rebase(repoPath string) error {
	_, err := git(repoPath, "rebase", "@{u}")
	return err
}

// Push pushes the repository to the remote
func Push(repoPath string) error {
	_, err := git(repoPath, "push")
	return err
}

// Stash stashes the current working tree
func StashSave(repoPath string) error {
	_, err := git(repoPath, "stash", "save")
	return err
}

// StashPop pops the last stash
func StashPop(repoPath string) error {
	_, err := git(repoPath, "stash", "pop")
	return err
}
