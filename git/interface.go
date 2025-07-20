package git

import "context"

// Git defines the interface for Git operations with clean data structures
type Git interface {
	// Repository checks
	IsRepo(path string) bool

	// Repository information
	GetBranchInfo(ctx context.Context, repoPath string) (BranchInfo, error)
	GetStatus(ctx context.Context, repoPath string) (StatusInfo, error)
	GetSyncInfo(ctx context.Context, repoPath string, branch string) (SyncInfo, error)

	// Repository operations
	Fetch(ctx context.Context, repoPath string, remote string) error
	Rebase(ctx context.Context, repoPath string) error
	Push(ctx context.Context, repoPath string) error
	StashSave(ctx context.Context, repoPath string) error
	StashPop(ctx context.Context, repoPath string) error
}

// New returns a new Git implementation
func New() Git {
	return &gitImpl{}
}