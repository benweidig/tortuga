package git

import (
	"bytes"
	"errors"
	"strings"
)

var (
	// ErrorAuthentication indicates that Git wasn't able to retrieve valid credentials
	ErrorAuthentication = errors.New("authentication failed")

	// ErrorNoUpstream indicates that the repository is local-only
	ErrorNoUpstream = errors.New("no upstream")
)

func isAuthError(stdErr string) bool {
	return strings.HasPrefix(stdErr, "fatal: could not read Username") ||
		strings.HasPrefix(stdErr, "fatal: could not read Password")
}

func isNoUpstreamError(stdErr string) bool {
	return strings.HasPrefix(stdErr, "fatal: no upstream")
}

// ConcretizeError parses the stdErr output to build a more meaningful error than what came back from git
func ConcretizeError(err error, stdErr bytes.Buffer) error {
	if err == nil {
		return err
	}

	errOut := stdErr.String()
	switch {
	case isAuthError(errOut):
		return ErrorAuthentication
	case isNoUpstreamError(errOut):
		return ErrorNoUpstream
	}

	return err
}
