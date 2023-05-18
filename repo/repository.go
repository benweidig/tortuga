package repo

import (
	"bufio"
	"path"
	"strings"

	"github.com/benweidig/tortuga/git"
)

// Repository represents Git repository, but only the currently checked out branch
type Repository struct {
	path string

	Name        string
	Branch      string
	Remote      string
	Changes     int
	stashed     bool
	Unversioned int
	Incoming    int
	Outgoing    int
	State       State
	Error       error
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
		r.withError(err).Branch = "???"
		return r, err
	}
	r.Branch = branch

	upstreamBranch, err := git.UpstreamBranch(r.path)
	if err != nil {
		r.withError(err)
		return r, err
	}
	r.Remote = strings.Split(upstreamBranch, "/")[0]

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
	status, err := git.Status(r.path)
	if err != nil {
		return r.withError(err).Error
	}
	scanner := bufio.NewScanner(&status)

	for scanner.Scan() {
		row := scanner.Text()
		if len(row) < 3 {
			continue
		}
		prefix := string(row[0:3])

		switch strings.TrimSpace(prefix) {
		case "M":
			r.Changes++
		case "A":
			r.Changes++
		case "D":
			r.Changes++
		case "R":
			r.Changes++
		case "C":
			r.Changes++
		case "U":
			r.Changes++
		case "??":
			r.Unversioned++
		}
	}

	err = git.Fetch(r.path, r.Remote)
	if err != nil {
		return r.withError(err).Error
	}

	incoming, err := git.Incoming(r.path, r.Branch)
	if err != nil {
		return r.withError(err).Error
	}
	r.Incoming = incoming

	outgoing, err := git.Outgoing(r.path, r.Branch)
	if err != nil {
		return r.withError(err).Error
	}
	r.Outgoing = outgoing

	r.State = StateRemoteFetched

	return nil
}

// Sync stashes, rebases, pushs and unstashes the Repository
func (r *Repository) Sync(incomingOnly bool) error {
	if r.State == StateError {
		return nil
	}

	errorReturn := func(err error) error {
		if r.stashed {
			git.StashPop(r.path)
		}
		return r.withError(err).Error
	}

	if r.Changes > 0 {
		err := git.Stash(r.path)
		if err != nil {
			return errorReturn(err)
		}
		r.stashed = true
	}

	if r.Incoming > 0 {
		err := git.Rebase(r.path)
		if err != nil {
			return errorReturn(err)
		}
	}

	if !incomingOnly && r.Outgoing > 0 {
		err := git.Push(r.path)
		if err != nil {
			return errorReturn(err)
		}
	}

	if r.stashed {
		err := git.StashPop(r.path)
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
