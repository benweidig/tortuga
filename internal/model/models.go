// Package model defines the shared data types used across all packages.
package model

// FetchStatus tracks the lifecycle of a repo during the fetch phase (Phase 1).
type FetchStatus int

const (
	FetchPending FetchStatus = iota // not yet started
	FetchRunning                    // git fetch in progress
	FetchDone                       // fetch complete; Ahead/Behind/Status populated
	FetchError                      // fetch or status check failed; see Repo.Err
)

// WorkStatus tracks the lifecycle of a repo during the sync phase (Phase 2).
type WorkStatus int

const (
	WorkPending    WorkStatus = iota // not yet started
	WorkStashing                     // running git stash
	WorkPulling                      // running git rebase @{u}
	WorkPushing                      // running git push
	WorkUnstashing                   // running git stash pop
	WorkDone                         // all steps completed successfully
	WorkError                        // a step failed; see Repo.Err
)

// GitErrorKind categorises a git failure so the UI can decide how to present it.
type GitErrorKind int

const (
	ErrAuthFailed     GitErrorKind = iota // credentials rejected or missing
	ErrRepoNotFound                       // remote URL not found (404/not found)
	ErrNoNetwork                          // DNS/TCP failure reaching remote
	ErrNotFastForward                     // rebase cannot proceed; branches diverged
	ErrStashConflict                      // stash pop produced merge conflicts
	ErrPushRejected                       // remote rejected the push (non-fast-forward)
	ErrNoUpstream                         // no tracking branch configured
	ErrUnknown                            // unrecognised error; Message holds raw stderr
)

// GitError is a structured git failure. It implements the error interface so it
// can be stored in standard error variables while also carrying a Kind for
// programmatic handling.
type GitError struct {
	Kind    GitErrorKind
	Message string // human-readable description, safe to show in the UI
}

func (e *GitError) Error() string {
	return e.Message
}

// StatusInfo holds the working-tree dirty state of a repo.
type StatusInfo struct {
	ChangedFiles     int // tracked files with uncommitted changes
	UnversionedFiles int // files not tracked by git
}

// IsDirty reports whether the working tree has any uncommitted changes.
func (s StatusInfo) IsDirty() bool {
	return s.ChangedFiles > 0 || s.UnversionedFiles > 0
}

// Repo is the central data structure passed between packages. It is always
// copied by value; mutations are made on the copy and sent back via callbacks
// or channels so the UI slice stays consistent.
type Repo struct {
	Path   string
	Name   string // basename of Path
	Branch string

	FetchStatus FetchStatus
	WorkStatus  WorkStatus

	Ahead      int  // local commits not on remote (need push)
	Behind     int  // remote commits not local (need pull)
	NoUpstream bool // true when no remote tracking branch is configured
	Status     StatusInfo
	Err        *GitError
}
