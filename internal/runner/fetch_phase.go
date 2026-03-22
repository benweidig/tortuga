// Package runner executes git operations concurrently. It owns all goroutines
// and semaphore logic, and reports progress through a serial onUpdate callback
// so callers (the ui package) never need to synchronise against each other.
package runner

import (
	"sync"

	"github.com/benweidig/tortuga/internal/git"
	"github.com/benweidig/tortuga/internal/model"
)

// repoUpdate carries a single state snapshot from a worker goroutine to the
// serial drain loop. Using an index rather than a pointer keeps Repo as a
// plain value type and avoids any data races on the slice.
type repoUpdate struct {
	index int
	repo  model.Repo
}

// RunFetchPhase concurrently fetches status for every repo and calls onUpdate after
// each state change. It blocks until all repos are done. onUpdate is always
// called serially — the caller does not need to synchronise it.
func RunFetchPhase(repos []model.Repo, sem chan struct{}, onUpdate func(idx int, repo model.Repo)) {
	updates := make(chan repoUpdate, len(repos))

	var wg sync.WaitGroup
	for i, repo := range repos {
		wg.Go(func() {
			sem <- struct{}{}
			defer func() { <-sem }()

			repo.FetchStatus = model.FetchRunning
			updates <- repoUpdate{i, repo}

			if gitErr := git.Fetch(repo.Path); gitErr != nil {
				repo.FetchStatus = model.FetchError
				repo.Err = gitErr
				updates <- repoUpdate{i, repo}
				return
			}

			ahead, behind, hasUpstream, err := git.AheadBehind(repo.Path)
			if err != nil {
				repo.FetchStatus = model.FetchError
				repo.Err = &model.GitError{Kind: model.ErrUnknown, Message: err.Error()}
				updates <- repoUpdate{i, repo}
				return
			}
			repo.Ahead = ahead
			repo.Behind = behind
			repo.NoUpstream = !hasUpstream

			status, err := git.GetStatus(repo.Path)
			if err != nil {
				repo.FetchStatus = model.FetchError
				repo.Err = &model.GitError{Kind: model.ErrUnknown, Message: err.Error()}
				updates <- repoUpdate{i, repo}
				return
			}
			repo.Status = status
			repo.FetchStatus = model.FetchDone
			updates <- repoUpdate{i, repo}
		})
	}

	go func() {
		wg.Wait()
		close(updates)
	}()

	for u := range updates {
		onUpdate(u.index, u.repo)
	}
}
