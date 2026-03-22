package git

import (
	"strings"

	"github.com/benweidig/tortuga/internal/model"
)

// ClassifyError maps git stderr output to a GitError with a human-readable message.
// step is the operation name ("fetch", "pull", "push", "stash", "stash pop") and is
// used to disambiguate patterns that can appear in multiple contexts.
func ClassifyError(stderr, step string) *model.GitError {
	s := stderr

	if containsAny(s, "Authentication failed", "Permission denied (publickey)", "could not read Username") {
		return &model.GitError{
			Kind:    model.ErrAuthFailed,
			Message: "authentication failed — check credentials",
		}
	}

	if containsAny(s, "Repository not found") || (strings.Contains(s, "repository '") && strings.Contains(s, "not found")) {
		return &model.GitError{
			Kind:    model.ErrRepoNotFound,
			Message: "remote repository not found",
		}
	}

	if containsAny(s, "Could not resolve host", "Connection timed out", "unable to connect") {
		return &model.GitError{
			Kind:    model.ErrNoNetwork,
			Message: "could not reach remote",
		}
	}

	if strings.Contains(s, "Not possible to fast-forward") {
		return &model.GitError{
			Kind:    model.ErrNotFastForward,
			Message: "cannot fast-forward — branches have diverged",
		}
	}

	if step == "stash pop" && strings.Contains(s, "CONFLICT") {
		return &model.GitError{
			Kind:    model.ErrStashConflict,
			Message: "stash pop conflict — changes preserved in stash",
		}
	}

	if containsAny(s, "[rejected]", "Updates were rejected") {
		return &model.GitError{
			Kind:    model.ErrPushRejected,
			Message: "push rejected — pull first",
		}
	}

	if containsAny(s, "no upstream branch", "no tracking information") {
		return &model.GitError{
			Kind:    model.ErrNoUpstream,
			Message: "no upstream branch configured",
		}
	}

	// Fallback: use first non-empty line of stderr
	msg := firstLine(stderr)

	return &model.GitError{
		Kind:    model.ErrUnknown,
		Message: msg,
	}
}

func containsAny(s string, substrings ...string) bool {
	for _, sub := range substrings {
		if strings.Contains(s, sub) {
			return true
		}
	}

	return false
}

func firstLine(s string) string {
	for line := range strings.SplitSeq(s, "\n") {
		line = strings.TrimSpace(line)

		if line != "" {
			return line
		}
	}

	return s
}
