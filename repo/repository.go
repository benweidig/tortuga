package repo

import (
	"context"
	"path"

	"github.com/benweidig/tortuga/git"
)

// Repository represents Git repository, but only the currently checked out branch
type Repository struct {
	path string
	git  git.Git

	Name   string
	Branch string
	Remote string
	State  State
	Error  error

	Incoming int
	Outgoing int

	Changes     int
	Unversioned int

	stashed bool
}

// NewRepository creates a bare Repository construct containing the minimum for initial display
func NewRepository(repoPath string) (*Repository, error) {
	return NewRepositoryWithGit(repoPath, git.New())
}

// NewRepositoryWithGit creates a repository with a specific Git implementation
func NewRepositoryWithGit(repoPath string, gitImpl git.Git) (*Repository, error) {
	r := &Repository{
		Name:  path.Base(repoPath),
		path:  repoPath,
		git:   gitImpl,
		State: StateNone,
	}

	ctx := context.Background()
	branchInfo, err := r.git.GetBranchInfo(ctx, r.path)
	if err != nil {
		r.withError(err).Branch = "???"
		return r, err
	}
	r.Branch = branchInfo.LocalBranch
	r.Remote = branchInfo.Remote

	return r, nil
}

func (r *Repository) withError(err error) *Repository {
	if err == nil {
		return nil
	}
	r.State = StateError
	r.Error = err
	return r
}

// Update analyzes the current working tree and fetches remote changes
func (r *Repository) Update() error {
	if r.State == StateError {
		return nil
	}

	ctx := context.Background()

	// Get status information
	statusInfo, err := r.git.GetStatus(ctx, r.path)
	if err != nil {
		return r.withError(err).Error
	}
	r.Changes = statusInfo.ChangedFiles
	r.Unversioned = statusInfo.UnversionedFiles

	// Fetch from remote
	err = r.git.Fetch(ctx, r.path, r.Remote)
	if err != nil {
		return r.withError(err).Error
	}

	// Get sync information
	syncInfo, err := r.git.GetSyncInfo(ctx, r.path, r.Branch)
	if err != nil {
		return r.withError(err).Error
	}
	r.Incoming = syncInfo.IncomingCommits
	r.Outgoing = syncInfo.OutgoingCommits

	r.State = StateRemoteFetched

	return nil
}

// Sync stashes, rebases, pushs and unstashes the Repository
func (r *Repository) Sync(incomingOnly bool) error {
	if r.State == StateError {
		return nil
	}

	ctx := context.Background()

	errorReturn := func(err error) error {
		if r.stashed {
			r.git.StashPop(ctx, r.path)
		}
		return r.withError(err).Error
	}

	if r.Changes > 0 {
		err := r.git.StashSave(ctx, r.path)
		if err != nil {
			return errorReturn(err)
		}
		r.stashed = true
	}

	if r.Incoming > 0 {
		err := r.git.Rebase(ctx, r.path)
		if err != nil {
			return errorReturn(err)
		}
	}

	if !incomingOnly && r.Outgoing > 0 {
		err := r.git.Push(ctx, r.path)
		if err != nil {
			return errorReturn(err)
		}
	}

	if r.stashed {
		err := r.git.StashPop(ctx, r.path)
		if err != nil {
			return r.withError(err).Error
		}
	}

	r.State = StateSynced

	return nil
}

// NeedsSync returns true if there are any changes that needs to be synced
func (r *Repository) NeedsSync() bool {
	return r.Incoming > 0 || r.Outgoing > 0
}

// ErrorCount return the total count of repositories with errors
func ErrorCount(r []*Repository) int {
	count := 0
	for _, repo := range r {
		if repo.State == StateError {
			count++
		}
	}
	return count
}

func (r *Repository) Noop() bool {
	return r.Changes == 0 && r.Unversioned == 0 && r.Incoming == 0 && r.Outgoing == 0
}
