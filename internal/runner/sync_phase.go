package runner

import (
	"sync"

	"github.com/benweidig/tortuga/internal/git"
	"github.com/benweidig/tortuga/internal/model"
)

// RunSyncPhase concurrently executes sync work for every repo and calls onUpdate
// after each step. push controls whether ahead commits are pushed.
// Blocks until all repos are done. onUpdate is always called serially.
func RunSyncPhase(repos []model.Repo, push bool, sem chan struct{}, onUpdate func(idx int, repo model.Repo)) {
	updates := make(chan repoUpdate, len(repos))

	var wg sync.WaitGroup
	for i, repo := range repos {
		wg.Go(func() {
			sem <- struct{}{}
			defer func() { <-sem }()

			runWork(i, repo, push, updates)
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

func runWork(idx int, r model.Repo, push bool, updates chan<- repoUpdate) {
	stashed := false

	if r.Status.IsDirty() {
		r.WorkStatus = model.WorkStashing
		updates <- repoUpdate{idx, r}

		if err := git.Stash(r.Path); err != nil {
			r.WorkStatus = model.WorkError
			r.Err = &model.GitError{Kind: model.ErrUnknown, Message: err.Error()}
			updates <- repoUpdate{idx, r}
			return
		}
		stashed = true
	}

	if r.Behind > 0 {
		r.WorkStatus = model.WorkPulling
		updates <- repoUpdate{idx, r}

		if gitErr := git.Rebase(r.Path); gitErr != nil {
			r.WorkStatus = model.WorkError
			r.Err = gitErr
			updates <- repoUpdate{idx, r}
			if stashed {
				unstash(idx, r, updates)
			}
			return
		}
	}

	if r.Ahead > 0 && push {
		r.WorkStatus = model.WorkPushing
		updates <- repoUpdate{idx, r}

		if gitErr := git.Push(r.Path); gitErr != nil {
			r.WorkStatus = model.WorkError
			r.Err = gitErr
			updates <- repoUpdate{idx, r}
			if stashed {
				unstash(idx, r, updates)
			}
			return
		}
	}

	if stashed {
		unstash(idx, r, updates)
		return
	}

	r.WorkStatus = model.WorkDone
	updates <- repoUpdate{idx, r}
}

func unstash(idx int, r model.Repo, updates chan<- repoUpdate) {
	// Snapshot the state before transitioning to WorkUnstashing so we can
	// restore it if stash pop succeeds but the earlier operation had failed.
	// Without this, a successful pop would overwrite WorkError with WorkDone,
	// hiding the failure in the status cell.
	prevStatus := r.WorkStatus
	prevErr := r.Err

	r.WorkStatus = model.WorkUnstashing
	updates <- repoUpdate{idx, r}

	if gitErr := git.StashPop(r.Path); gitErr != nil {
		r.WorkStatus = model.WorkError
		r.Err = gitErr
	} else if prevStatus == model.WorkError {
		r.WorkStatus = model.WorkError
		r.Err = prevErr
	} else {
		r.WorkStatus = model.WorkDone
	}
	updates <- repoUpdate{idx, r}
}
