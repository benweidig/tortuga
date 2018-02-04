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
	Status   Status
	Incoming int
	Outgoing int
	Checked  bool
}

func NewRepository(repoPath string) (Repository, error) {
	r := Repository{
		Name: path.Base(repoPath),
		path: repoPath,
	}

	branch, stdErr, err := git.CurrentBranch(r.path)
	if err != nil {
		fmt.Println(err)
		log.Fatal(stdErr.String())
		return r, err
	}

	r.Branch = branch

	return r, nil
}

func (r *Repository) Update(localOnly bool) error {
	var err error
	if localOnly == false {
		_, err := git.FetchAll(r.path)
		if err != nil {
			return err
		}

		err = r.updateStatus()
		if err != nil {
			return err
		}
	}

	err = r.updateIncoming()
	if err != nil {
		return err
	}

	err = r.updateOutgoing()
	if err != nil {
		return err
	}

	r.Checked = true

	return nil
}

func (r *Repository) updateIncoming() error {
	count, _, err := git.Incoming(r.path, r.Branch)
	if err != nil {
		return err
	}

	r.Incoming = count

	return nil
}

func (r *Repository) updateOutgoing() error {
	count, _, err := git.Outgoing(r.path, r.Branch)
	if err != nil {
		return err
	}

	r.Outgoing = count

	return nil
}

func (r *Repository) updateStatus() error {
	out, _, err := git.Status(r.path)
	if err != nil {
		return err
	}

	r.Status = NewStatus(out)

	return nil
}

func (r Repository) Stash() error {
	if r.Status.Stashable == 0 {
		return nil
	}

	_, err := git.Stash(r.path)
	if err != nil {
		return err
	}

	return err
}

func (r Repository) Unstash() error {
	if r.Status.Stashable == 0 {
		return nil
	}

	_, err := git.PopStash(r.path)
	if err != nil {
		return err
	}

	return nil
}

func (r Repository) Pull() error {
	_, err := git.Pull(r.path)
	if err != nil {
		fmt.Printf("Couldn't pull %s [%s]: %s", r.Name, r.Branch, err)
		return err
	}

	return err
}

func (r Repository) PullRebase() error {
	_, err := git.PullRebase(r.path)

	if err != nil {
		fmt.Printf("Couldn't pull/rebase %s [%s]: %s", r.Name, r.Branch, err)
		return err
	}

	return err
}

func (r Repository) Push() error {
	_, err := git.Push(r.path)
	return err
}
