package git

// StatusInfo represents the status of a git repository
type StatusInfo struct {
	ChangedFiles    int
	UnversionedFiles int
	IsDirty         bool
}

// BranchInfo contains information about repository branches
type BranchInfo struct {
	LocalBranch    string
	UpstreamBranch string
	Remote         string
}

// SyncInfo represents incoming/outgoing commit information
type SyncInfo struct {
	IncomingCommits int
	OutgoingCommits int
	NeedsSync       bool
}

// RepositoryInfo combines all repository information
type RepositoryInfo struct {
	Path   string
	Branch BranchInfo
	Status StatusInfo
	Sync   SyncInfo
}