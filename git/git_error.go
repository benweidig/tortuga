package git

import (
	"bytes"
	"strings"
)

// GitError is a wrapper around an error occurred running the git executable
type GitError struct {
	Cause   error
	StdErr  string
	message string
}

func (e *GitError) Error() string {
	return e.message
}

// Unwrap returns the underlying error for error chain support
func (e *GitError) Unwrap() error {
	return e.Cause
}

func isAuthError(stderr string) bool {
	return strings.HasPrefix(stderr, "fatal: could not read Username") ||
		strings.HasPrefix(stderr, "fatal: could not read Password")
}

func isNoUpstreamError(stderr string) bool {
	return strings.HasPrefix(stderr, "fatal: no upstream")
}

func wrapError(err error, stdErr bytes.Buffer) *GitError {
	if err == nil {
		return nil
	}

	stderrStr := stdErr.String()
	var message string

	switch {
	case isAuthError(stderrStr):
		message = "auth error"
	case isNoUpstreamError(stderrStr):
		message = "no upstream"
	default:
		message = "error"
	}

	return &GitError{
		Cause:   err,
		StdErr:  stderrStr,
		message: message,
	}
}