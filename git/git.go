// Run git commands against a specific worktree
package git

import (
	"bufio"
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// Runs a git command with the specified args against a path
func run(repoPath string, args ...string) (bytes.Buffer, bytes.Buffer, error) {
	// Combine args and build command
	args = append([]string{"-C", repoPath}, args...)
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
func CurrentBranch(repoPath string) (string, bytes.Buffer, error) {
	branchBuffer, stdErr, err := run(repoPath, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", stdErr, err
	}

	// We have to sanitize the output for easier usage
	branch := branchBuffer.String()
	branch = strings.TrimSuffix(branch, "\n")
	return branch, stdErr, err
}

// Fetches all remotes of the repository
func FetchAll(repoPath string) (bytes.Buffer, error) {
	_, stdErr, err := run(repoPath, "fetch", "--all")
	return stdErr, err
}

// Shows the working tree status
func Status(repoPath string) (bytes.Buffer, bytes.Buffer, error) {
	return run(repoPath, "status", "--porcelain")
}

// Returns the count of incomming changes between the local head and upstream
func Incoming(repoPath string, branch string) (int, bytes.Buffer, error) {
	rangeSpecifier := fmt.Sprintf("HEAD..%s@{upstream}", branch)
	return commitDiff(repoPath, rangeSpecifier)
}

// Returns the count of outgoing changes between the local head and upstrea
func Outgoing(repoPath string, branch string) (int, bytes.Buffer, error) {
	rangeSpecifier := fmt.Sprintf("%s@{upstream}..HEAD", branch)
	return commitDiff(repoPath, rangeSpecifier)
}

func commitDiff(repoPath string, rangeSpecifier string) (int, bytes.Buffer, error) {
	stdOut, stdErr, err := run(repoPath, "rev-list", rangeSpecifier)
	if err != nil {
		return 0, stdErr, err
	}

	scanner := bufio.NewScanner(&stdOut)
	count := 0
	for scanner.Scan() {
		count += 1
	}

	if err := scanner.Err(); err != nil {
		return 0, stdErr, err
	}
	return count, stdErr, nil
}

// Stash the changes in a dirty directory away
func Stash(repoPath string) (bytes.Buffer, error) {
	_, stdErr, err := run(repoPath, "stash")
	return stdErr, err
}

// Remove top-most stashed state from the stash list and apply it on top of the current working tree state
func PopStash(repoPath string) (bytes.Buffer, error) {
	_, stdErr, err := run(repoPath, "stash", "pop")
	return stdErr, err
}

// Fetch from and integrate changes to the current working tree state by merge (default pull)
func Pull(repoPath string) (bytes.Buffer, error) {
	_, stdErr, err := run(repoPath, "pull")
	return stdErr, err
}

// Fetch from and integrate changes to the current working tree state by rebase
func PullRebase(repoPath string) (bytes.Buffer, error) {
	_, stdErr, err := run(repoPath, "pull", "--rebase")
	return stdErr, err
}

// Update remote refs using local refs, while sending objects becessary to complete given refs
func Push(repoPath string) (bytes.Buffer, error) {
	_, stdErr, err := run(repoPath, "push")
	return stdErr, err
}
