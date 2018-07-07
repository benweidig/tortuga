package repo

import (
	"path"
	"strings"

	"github.com/benweidig/tortuga/git"
)

// Repository represents Git repository, but only the currently checked out branch
type Repository struct {
	path string

	Name         string
	Branch       string
	Remote       string
	LocalChanges Changes
	Incoming     int
	Outgoing     int
	State        State
	Error        error
}

// NewRepository creates a bare Repository construct containing the minimum for initial display
func NewRepository(repoPath string) (*Repository, error) {
	r := &Repository{
		Name:  path.Base(repoPath),
		path:  repoPath,
		State: StateNone,
	}

	branch, err := git.LocalBranch(r.path)
	if err != nil {
		r.registerError(err)
		r.Branch = "???"
		return r, err
	}
	r.Branch = branch

	upstreamBranch, err := git.UpstreamBranch(r.path)
	if err != nil {
		r.registerError(err)
		return r, err
	}
	r.Remote = strings.Split(upstreamBranch, "/")[0]

	return r, nil
}

func (r *Repository) registerError(err error) {
	r.State = StateError
	r.Error = err
}

// UpdateChanges gets and sets the current changes and number of incoming/outgoing of a Repository.
// If localOnly is true no fetching fo the remote will occur.
func (r *Repository) UpdateChanges(localOnly bool) error {
	if r.State == StateError {
		return nil
	}

	if localOnly == false {
		err := git.Fetch(r.path, r.Remote)
		if err != nil {
			r.registerError(err)
			return err
		}
	}

	status, err := git.Status(r.path)
	if err != nil {
		r.State = StateError
		r.Error = err
		return err
	}

	r.LocalChanges = NewChanges(status)

	incoming, err := git.Incoming(r.path, r.Branch)
	if err != nil {
		r.registerError(err)
		return err
	}
	r.Incoming = incoming

	outgoing, err := git.Outgoing(r.path, r.Branch)
	if err != nil {
		r.registerError(err)
		return err
	}
	r.Outgoing = outgoing

	r.State = StateChangesUpdated

	return nil
}

// Sync stashes, rebases, pushs and unstashes the Repository
func (r *Repository) Sync() error {
	if r.State >= StateError {
		return nil
	}

	if r.LocalChanges.Stashable > 0 {
		err := git.Stash(r.path)
		if err != nil {
			r.registerError(err)
			return err
		}
	}

	if r.Incoming > 0 {
		err := git.Rebase(r.path)
		if err != nil {
			r.registerError(err)
			return err
		}
	}

	if r.Outgoing > 0 {
		err := git.Push(r.path)
		if err != nil {
			r.registerError(err)
			return err
		}
	}

	if r.LocalChanges.Stashable > 0 {
		err := git.StashPop(r.path)
		if err != nil {
			r.registerError(err)
			return err
		}
	}

	r.State = StateSynced

	return nil
}

func (r *Repository) IsDirty() bool {
	return r.Incoming > 0 || r.Outgoing > 0
}
