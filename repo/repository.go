package repo

import (
	"fmt"
	"log"
	"path"

	"github.com/benweidig/tortuga/git"
)

type Repository struct {
	path string

	Name       string
	RemoteUrls RemoteUrls
	Branch     string
	Changes    Changes
	Incoming   int
	Outgoing   int
	Checked    bool
}

func NewRepository(repoPath string) (Repository, error) {
	r := Repository{
		Name:    path.Base(repoPath),
		path:    repoPath,
		Checked: false,
	}

	branch, stdErr, err := git.CurrentBranch(r.path)
	if err != nil {
		fmt.Println(err)
		log.Fatal(stdErr.String())
	}

	r.Branch = branch

	remoteUrlsBuffer, stdErr, err := git.RemoteVerbose(r.path)
	if err != nil {
		fmt.Println(err)
		log.Fatal(stdErr.String())
	}
	remoteUrls, err := NewRemoteUrls(remoteUrlsBuffer)
	if err != nil {
		fmt.Println(err)
		log.Fatal(stdErr.String())
	}
	r.RemoteUrls = remoteUrls

	return r, nil
}

func (r *Repository) Update(localOnly bool) error {
	var err error
	if localOnly == false {
		_, err := git.FetchAll(r.path)
		if err != nil {
			return err
		}
	}

	status, _, err := git.Status(r.path)
	if err != nil {
		return err
	}

	r.Changes = NewChanges(status)

	incoming, _, err := git.Incoming(r.path, r.Branch)
	if err != nil {
		return err
	}
	r.Incoming = incoming

	outgoing, _, err := git.Outgoing(r.path, r.Branch)
	if err != nil {
		return err
	}
	r.Outgoing = outgoing

	r.Checked = true

	return nil
}

func (r Repository) Sync() error {
	if r.Changes.Stashable > 0 {
		_, err := git.Stash(r.path)
		if err != nil {
			return err
		}
	}

	if r.Incoming > 0 {
		_, err := git.PullRebase(r.path)

		if err != nil {
			fmt.Printf("Couldn't pull/rebase %s [%s]: %s", r.Name, r.Branch, err)
			return err
		}
	}

	if r.Changes.Stashable > 0 {
		_, err := git.PopStash(r.path)
		if err != nil {
			return err
		}
	}

	if r.Outgoing > 0 {
		_, err := git.Push(r.path)
		if err != nil {
			return err
		}
	}

	return nil
}
