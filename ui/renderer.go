package ui

import (
	"context"
	"sync"
	"time"

	"github.com/benweidig/tortuga/repo"
)

// RenderRequest represents a request to render repository status
type RenderRequest struct {
	IncomingOnly bool
}

// UIRenderer handles all UI rendering through channels to eliminate synchronization issues
type UIRenderer interface {
	// Start begins the rendering goroutine - must be called before any rendering
	Start(ctx context.Context) error
	
	// Stop gracefully shuts down the renderer
	Stop() error
	
	// RequestRender sends a render request (non-blocking)
	RequestRender(incomingOnly bool)
	
	// RenderProgress creates a progress callback for repository operations
	RenderProgress(incomingOnly bool) func()
	
	// InitialRender performs the first render synchronously
	InitialRender(incomingOnly bool)
}

// channelRenderer implements UIRenderer using channels
type channelRenderer struct {
	manager        repo.RepositoryManager
	writer         *StdoutWriter
	renderRequests chan RenderRequest
	done           chan struct{}
	started        bool
	wg             sync.WaitGroup

	// Rate limiting
	minInterval time.Duration
	lastRender  time.Time
}

// NewUIRenderer creates a new channel-based UI renderer
func NewUIRenderer(manager repo.RepositoryManager, writer *StdoutWriter) UIRenderer {
	return &channelRenderer{
		manager:        manager,
		writer:         writer,
		renderRequests: make(chan RenderRequest, 100), // Buffered to prevent blocking
		done:           make(chan struct{}),
		minInterval:    50 * time.Millisecond, // Prevent excessive renders
	}
}

// Start begins the rendering goroutine
func (r *channelRenderer) Start(ctx context.Context) error {
	if r.started {
		return nil
	}
	r.started = true
	r.wg.Add(1)
	go r.renderLoop(ctx)
	return nil
}

// Stop gracefully shuts down the renderer, blocking until the goroutine exits
// and any pending final render is complete.
func (r *channelRenderer) Stop() error {
	if !r.started {
		return nil
	}
	close(r.done)
	r.wg.Wait()
	r.started = false
	return nil
}

// RequestRender sends a render request (non-blocking)
func (r *channelRenderer) RequestRender(incomingOnly bool) {
	if !r.started {
		return
	}
	
	select {
	case r.renderRequests <- RenderRequest{IncomingOnly: incomingOnly}:
		// Request sent successfully
	default:
		// Channel full, skip this render (rate limiting)
	}
}

// RenderProgress creates a progress callback for repository operations
func (r *channelRenderer) RenderProgress(incomingOnly bool) func() {
	return func() {
		r.RequestRender(incomingOnly)
	}
}

// InitialRender performs the first render synchronously
func (r *channelRenderer) InitialRender(incomingOnly bool) {
	r.performRender(incomingOnly)
}

// renderLoop is the main rendering goroutine
func (r *channelRenderer) renderLoop(ctx context.Context) {
	defer r.wg.Done()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	var pendingRender *RenderRequest

	for {
		select {
		case <-ctx.Done():
			return

		case <-r.done:
			// Drain any queued requests and do one final render
			for {
				select {
				case req := <-r.renderRequests:
					pendingRender = &req
				default:
					if pendingRender != nil {
						r.performRender(pendingRender.IncomingOnly)
					}
					return
				}
			}

		case req := <-r.renderRequests:
			pendingRender = &req

		case <-ticker.C:
			if pendingRender != nil && time.Since(r.lastRender) >= r.minInterval {
				r.performRender(pendingRender.IncomingOnly)
				pendingRender = nil
			}
		}
	}
}

// performRender executes the actual rendering
func (r *channelRenderer) performRender(incomingOnly bool) {
	r.lastRender = time.Now()
	
	// Reset the writer and render fresh content
	r.writer.Reset()
	
	repos := r.manager.GetRepositories()
	WriteRepositoryStatus(r.writer, repos, incomingOnly)
	
	r.writer.Flush()
}