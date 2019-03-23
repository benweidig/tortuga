package git

import (
	"bytes"
	"strings"
)

// ExternalError is a wrapper around an error occured running the git exectuable
type ExternalError struct {
	Cause   error
	StdOut  string
	StdErr  string
	message string
}

func (e *ExternalError) Error() string {
	if e == nil {
		return ""
	}
	return e.message
}

func isAuthError(ge *ExternalError) bool {
	return strings.HasPrefix(ge.StdErr, "fatal: could not read Username") ||
		strings.HasPrefix(ge.StdErr, "fatal: could not read Password")
}

func isNoUpstreamError(ge *ExternalError) bool {
	return strings.HasPrefix(ge.StdErr, "fatal: no upstream")
}

func wrapError(err error, stdOut bytes.Buffer, stdErr bytes.Buffer) *ExternalError {
	if err == nil {
		return nil
	}

	ge := &ExternalError{
		Cause:  err,
		StdOut: stdOut.String(),
		StdErr: stdErr.String(),
	}

	switch {
	case isAuthError(ge):
		ge.message = "auth error"
	case isNoUpstreamError(ge):
		ge.message = "no upstream"
	default:
		ge.message = "error"
	}

	return ge
}
