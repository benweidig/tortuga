package git

import (
	"bufio"
	"context"
	"strings"
)

// gitImpl implements the Git interface using the existing git functions
type gitImpl struct{}

// IsRepo checks if a path is a git repository
func (g *gitImpl) IsRepo(path string) bool {
	return IsRepo(path)
}

// GetBranchInfo retrieves branch information for a repository
func (g *gitImpl) GetBranchInfo(ctx context.Context, repoPath string) (BranchInfo, error) {
	var info BranchInfo

	localBranch, err := LocalBranch(ctx, repoPath)
	if err != nil {
		return info, err
	}
	info.LocalBranch = localBranch

	upstreamBranch, err := UpstreamBranch(ctx, repoPath)
	if err != nil {
		return info, err
	}
	info.UpstreamBranch = upstreamBranch

	// Extract remote name from upstream branch (e.g., "origin/main" -> "origin")
	if parts := strings.Split(upstreamBranch, "/"); len(parts) > 0 {
		info.Remote = parts[0]
	}

	return info, nil
}

// GetStatus retrieves status information for a repository
func (g *gitImpl) GetStatus(ctx context.Context, repoPath string) (StatusInfo, error) {
	var info StatusInfo

	status, err := Status(ctx, repoPath)
	if err != nil {
		return info, err
	}

	scanner := bufio.NewScanner(&status)
	for scanner.Scan() {
		row := scanner.Text()
		if len(row) < 3 {
			continue
		}

		statusCode := strings.TrimSpace(row[0:3])
		if len(statusCode) == 0 {
			continue
		}

		switch statusCode[0] {
		case 'M', 'T', 'A', 'D', 'R', 'C', 'U':
			info.ChangedFiles++
		case '?':
			info.UnversionedFiles++
		}
	}

	info.IsDirty = info.ChangedFiles > 0 || info.UnversionedFiles > 0

	return info, nil
}

// GetSyncInfo retrieves sync information (incoming/outgoing commits)
func (g *gitImpl) GetSyncInfo(ctx context.Context, repoPath string, branch string) (SyncInfo, error) {
	var info SyncInfo

	incoming, err := Incoming(ctx, repoPath, branch)
	if err != nil {
		return info, err
	}
	info.IncomingCommits = incoming

	outgoing, err := Outgoing(ctx, repoPath, branch)
	if err != nil {
		return info, err
	}
	info.OutgoingCommits = outgoing

	info.NeedsSync = info.IncomingCommits > 0 || info.OutgoingCommits > 0

	return info, nil
}

// Fetch fetches from the specified remote
func (g *gitImpl) Fetch(ctx context.Context, repoPath string, remote string) error {
	return Fetch(ctx, repoPath, remote)
}

// Rebase rebases the current branch with upstream
func (g *gitImpl) Rebase(ctx context.Context, repoPath string) error {
	return Rebase(ctx, repoPath)
}

// Push pushes the current branch to remote
func (g *gitImpl) Push(ctx context.Context, repoPath string) error {
	return Push(ctx, repoPath)
}

// StashSave stashes current changes
func (g *gitImpl) StashSave(ctx context.Context, repoPath string) error {
	return StashSave(ctx, repoPath)
}

// StashPop pops the last stash
func (g *gitImpl) StashPop(ctx context.Context, repoPath string) error {
	return StashPop(ctx, repoPath)
}