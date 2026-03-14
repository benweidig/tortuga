package repo

import (
	"context"
	"errors"
	"os"
	"path"
	"sync"

	"github.com/benweidig/tortuga/git"
)

// ProgressCallback is called during repository operations to report progress
type ProgressCallback func()

// RepositoryManager handles a collection of repositories and their operations
type RepositoryManager interface {
	// Discovery
	Discover(basePath string) error
	GetRepositories() []*Repository
	Count() int

	// Operations
	UpdateAll(ctx context.Context, progressCallback ProgressCallback) error
	SyncAll(ctx context.Context, incomingOnly bool, progressCallback ProgressCallback) error

	// Aggregation
	TotalIncoming() int
	TotalOutgoing() int
	HasChangesToSync() bool
	ErrorCount() int
}

// repositoryManagerImpl implements the RepositoryManager interface
type repositoryManagerImpl struct {
	repositories []*Repository
	git          git.Git
	mu           sync.RWMutex
}

// NewManager creates a new repository manager
func NewManager() RepositoryManager {
	return &repositoryManagerImpl{
		git: git.New(),
	}
}

// Discover finds all repositories in the given path
func (m *repositoryManagerImpl) Discover(basePath string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.repositories = m.repositories[:0]

	if m.git.IsRepo(basePath) {
		r, err := NewRepositoryWithGit(basePath, m.git)
		if err != nil {
			return err
		}
		m.repositories = append(m.repositories, r)
		return nil
	}

	entries, err := os.ReadDir(basePath)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		entryPath := path.Join(basePath, entry.Name())
		if !m.git.IsRepo(entryPath) {
			continue
		}

		// Build repository. We ignore errors so all will be displayed
		r, _ := NewRepositoryWithGit(entryPath, m.git)
		m.repositories = append(m.repositories, r)
	}

	return nil
}

// GetRepositories returns a copy of the repositories slice for safe iteration
func (m *repositoryManagerImpl) GetRepositories() []*Repository {
	m.mu.RLock()
	defer m.mu.RUnlock()

	repos := make([]*Repository, len(m.repositories))
	copy(repos, m.repositories)
	return repos
}

// Count returns the number of repositories
func (m *repositoryManagerImpl) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.repositories)
}

// UpdateAll updates all repositories in parallel
func (m *repositoryManagerImpl) UpdateAll(ctx context.Context, progressCallback ProgressCallback) error {
	repos := m.GetRepositories()
	if len(repos) == 0 {
		return nil
	}

	// Initial progress callback
	if progressCallback != nil {
		progressCallback()
	}

	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		errs []error
	)

	for _, r := range repos {
		wg.Go(func() {
			if err := r.Update(ctx); err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
			}
			if progressCallback != nil {
				progressCallback()
			}
		})
	}

	wg.Wait()

	if progressCallback != nil {
		progressCallback()
	}

	return errors.Join(errs...)
}

// SyncAll syncs all repositories that need syncing in parallel
func (m *repositoryManagerImpl) SyncAll(ctx context.Context, incomingOnly bool, progressCallback ProgressCallback) error {
	repos := m.GetRepositories()
	if len(repos) == 0 {
		return nil
	}

	// Set sync states
	for _, r := range repos {
		if r.State == StateError {
			continue
		}
		if r.NeedsSync() {
			r.State = StateNeedsSync
		} else {
			r.State = StateNoSyncNeeded
		}
	}

	// Initial progress callback
	if progressCallback != nil {
		progressCallback()
	}

	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		errs []error
	)

	for _, r := range repos {
		wg.Go(func() {
			if r.State == StateNeedsSync {
				if err := r.Sync(ctx, incomingOnly); err != nil {
					mu.Lock()
					errs = append(errs, err)
					mu.Unlock()
				}
			}
			if progressCallback != nil {
				progressCallback()
			}
		})
	}

	wg.Wait()

	if progressCallback != nil {
		progressCallback()
	}

	return errors.Join(errs...)
}

// TotalIncoming returns the total number of incoming commits across all repositories
func (m *repositoryManagerImpl) TotalIncoming() int {
	repos := m.GetRepositories()
	total := 0
	for _, r := range repos {
		total += r.Incoming
	}
	return total
}

// TotalOutgoing returns the total number of outgoing commits across all repositories
func (m *repositoryManagerImpl) TotalOutgoing() int {
	repos := m.GetRepositories()
	total := 0
	for _, r := range repos {
		total += r.Outgoing
	}
	return total
}

// HasChangesToSync returns true if any repository needs syncing
func (m *repositoryManagerImpl) HasChangesToSync() bool {
	return m.TotalIncoming() > 0 || m.TotalOutgoing() > 0
}

// ErrorCount returns the number of repositories with errors
func (m *repositoryManagerImpl) ErrorCount() int {
	repos := m.GetRepositories()
	count := 0
	for _, r := range repos {
		if r.State == StateError {
			count++
		}
	}
	return count
}
