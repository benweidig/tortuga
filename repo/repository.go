package repo

import (
	"fmt"
	"log"
	"path"

	"github.com/benweidig/tortuga/git"
)

type Repository struct {
	path string

	Name     string
	Branch   string
	Changes  Changes
	Incoming int
	Outgoing int
	State    State
}

func NewRepository(repoPath string) (Repository, error) {
	r := Repository{
		Name:  path.Base(repoPath),
		path:  repoPath,
		State: StateNone,
	}

	branch, stdErr, err := git.CurrentBranch(r.path)
	if err != nil {
		fmt.Println(err)
		log.Fatal(stdErr.String())
	}

	r.Branch = branch

	return r, nil
}

// Updates a Repository with the current changes and number of incoming/outgoing.
// If localOnly is true no fetching fo the remote will occur.
func (r *Repository) Update(localOnly bool) error {
	if localOnly == false {
		_, err := git.FetchAll(r.path)
		if err != nil {
			r.State = StateError
			return err
		}
	}

	status, _, err := git.Status(r.path)
	if err != nil {
		r.State = StateError
		return err
	}

	r.Changes = NewChanges(status)

	incoming, _, err := git.Incoming(r.path, r.Branch)
	if err != nil {
		r.State = StateError
		return err
	}
	r.Incoming = incoming

	outgoing, _, err := git.Outgoing(r.path, r.Branch)
	if err != nil {
		r.State = StateError
		return err
	}
	r.Outgoing = outgoing

	r.State = StateUpdated

	return nil
}

func (r *Repository) Sync() error {
	if r.Changes.Stashable > 0 {
		_, err := git.Stash(r.path)
		if err != nil {
			r.State = StateError
			return err
		}
	}

	if r.Incoming > 0 {
		_, err := git.PullRebase(r.path)
		if err != nil {
			r.State = StateError
			return err
		}
	}

	if r.Outgoing > 0 {
		_, err := git.Push(r.path)
		if err != nil {
			r.State = StateError
			return err
		}
	}

	if r.Changes.Stashable > 0 {
		_, err := git.PopStash(r.path)
		if err != nil {
			r.State = StateError
			return err
		}
	}

	r.State = StateSynced

	return nil
}
