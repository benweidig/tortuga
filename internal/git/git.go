// Package git wraps git operations via os/exec. All functions operate on a
// repository identified by its directory path. No git library is used so that
// real-world edge cases (network errors, auth prompts, unusual git versions)
// surface as plain stderr text that ClassifyError can map to structured errors.
package git

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/benweidig/tortuga/internal/model"
)

// runGit executes a git command in the given directory and returns stdout and stderr.
func runGit(path string, args ...string) (stdout, stderr string, err error) {
	var outBuf, errBuf bytes.Buffer
	cmd := exec.Command("git", append([]string{"-C", path}, args...)...)
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err = cmd.Run()

	return strings.TrimSpace(outBuf.String()), strings.TrimSpace(errBuf.String()), err
}

// IsAvailable returns an error if git is not found in PATH.
func IsAvailable() error {
	_, err := exec.LookPath("git")
	return err
}

// LocalBranch returns the name of the currently checked-out branch. Returns an
// error when the repo is in detached-HEAD state (branch name would be "HEAD").
func LocalBranch(path string) (string, error) {
	stdout, _, err := runGit(path, "rev-parse", "--abbrev-ref", "HEAD")

	if err != nil {
		return "", err
	}

	if stdout == "HEAD" {
		return stdout, errors.New("detached HEAD")
	}

	return stdout, nil
}

// UpstreamBranch returns the full symbolic name of the upstream tracking branch
// (e.g. "origin/main"). Returns an error if no upstream is configured.
func UpstreamBranch(path string) (string, error) {
	stdout, _, err := runGit(path, "rev-parse", "--symbolic-full-name", "--abbrev-ref", "@{u}")

	if err != nil {
		return "", err
	}

	return stdout, nil
}

// Fetch runs "git fetch --quiet" against all remotes in the repo.
func Fetch(path string) *model.GitError {
	_, stderr, err := runGit(path, "fetch", "--quiet")

	if err != nil {
		return ClassifyError(stderr, "fetch")
	}

	return nil
}

// AheadBehind returns the number of local commits not yet pushed (ahead, via
// @{push}) and remote commits not yet pulled (behind, via @{upstream}).
// hasUpstream is false when no tracking branch is configured, in which case
// ahead and behind are both 0.
func AheadBehind(path string) (ahead, behind int, hasUpstream bool, err error) {
	if _, err := UpstreamBranch(path); err != nil {
		return 0, 0, false, nil
	}

	behind, err = revListCount(path, "HEAD..@{upstream}")
	if err != nil {
		return 0, 0, true, err
	}

	ahead, err = revListCount(path, "@{push}..HEAD")
	if err != nil {
		return 0, 0, true, err
	}

	return ahead, behind, true, nil
}

func revListCount(path, rangeSpec string) (int, error) {
	stdout, _, err := runGit(path, "rev-list", "--count", rangeSpec)

	if err != nil {
		return 0, err
	}

	n, err := strconv.Atoi(stdout)

	if err != nil {
		return 0, fmt.Errorf("unexpected rev-list output: %q", stdout)
	}

	return n, nil
}

// GetStatus returns parsed status information for the repo.
func GetStatus(path string) (model.StatusInfo, error) {
	var info model.StatusInfo

	stdout, _, err := runGit(path, "status", "--porcelain")
	if err != nil {
		return info, fmt.Errorf("git status: %w", err)
	}

	scanner := bufio.NewScanner(strings.NewReader(stdout))
	for scanner.Scan() {
		row := scanner.Text()

		if len(row) < 3 {
			continue
		}

		statusCode := strings.TrimSpace(row[0:3])

		if len(statusCode) == 0 {
			continue
		}

		switch statusCode[0] {
		case 'M', 'T', 'A', 'D', 'R', 'C', 'U':
			info.ChangedFiles++
		case '?':
			info.UnversionedFiles++
		}
	}

	return info, nil
}

// Stash runs "git stash --include-untracked", saving both tracked and
// untracked changes so the working tree is clean before a rebase.
func Stash(path string) error {
	_, stderr, err := runGit(path, "stash", "--include-untracked")

	if err != nil {
		return fmt.Errorf("git stash: %s", firstLine(stderr))
	}

	return nil
}

// Rebase runs "git rebase @{u}" to integrate upstream commits. Rebase is
// preferred over merge to keep a linear history; @{u} (the upstream tracking
// branch) is used directly so the remote ref is always current after a Fetch.
func Rebase(path string) *model.GitError {
	_, stderr, err := runGit(path, "rebase", "@{u}")

	if err != nil {
		return ClassifyError(stderr, "rebase")
	}

	return nil
}

// Push runs "git push" using the default remote and tracking branch.
func Push(path string) *model.GitError {
	_, stderr, err := runGit(path, "push")

	if err != nil {
		return ClassifyError(stderr, "push")
	}

	return nil
}

// StashPop restores the most recent stash entry. If a conflict occurs the
// working tree is left in a conflicted state and the stash entry is kept.
func StashPop(path string) *model.GitError {
	_, stderr, err := runGit(path, "stash", "pop")

	if err != nil {
		return ClassifyError(stderr, "stash pop")
	}

	return nil
}
