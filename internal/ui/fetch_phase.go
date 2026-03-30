package ui

import (
	"sync"
	"time"

	"github.com/benweidig/tortuga/internal/model"
	"github.com/benweidig/tortuga/internal/runner"
)

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func RunFetchPhase(repos []model.Repo, sem chan struct{}, renderer *Renderer) ([]model.Repo, error) {
	// mu guards both repos and all renderer calls. The spinner goroutine and the
	// runner callback run concurrently; without the mutex a ticker-driven partial
	// update could interleave with a callback-driven one, leaving the cursor in
	// the wrong place.
	var mu sync.Mutex
	spinnerIdx := 0

	done := make(chan struct{})
	var spinnerWg sync.WaitGroup
	spinnerWg.Go(func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				mu.Lock()
				spinnerIdx = (spinnerIdx + 1) % len(spinnerFrames)
				renderer.RenderTick(repos, spinnerFrames[spinnerIdx])
				mu.Unlock()
			case <-done:
				return
			}
		}
	})

	renderer.Render(repos, spinnerFrames[0])

	// runner.RunFetchPhase blocks until all repos are done and calls the callback
	// serially, so the callback itself needs no additional synchronisation beyond
	// the shared mu.
	runner.RunFetchPhase(repos, sem, func(idx int, repo model.Repo) {
		mu.Lock()
		repos[idx] = repo
		renderer.RenderRowUpdate(idx, repos, spinnerFrames[spinnerIdx])
		mu.Unlock()
	})

	close(done)
	// Wait for the spinner goroutine to fully exit before the final render.
	// Without this, the goroutine could fire one last tick after close(done)
	// returns but before renderer.Render below, overwriting the final frame.
	spinnerWg.Wait()

	renderer.RenderFull(repos)
	renderer.RenderErrors(repos)

	return repos, nil
}
