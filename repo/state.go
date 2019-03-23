package repo

// State represents representing the current state of a Repository in the Tortuga workflow
type State int

const (
	// StateNone is the initial state, no actual work is done so far
	StateNone State = iota

	// StateRemoteFetched is after fetching the remote
	StateRemoteFetched

	// StateNeedsSync means the repository has incoming/outgoing changes
	StateNeedsSync

	// StateNoSyncNeeded means the repository has no changes to be synced
	StateNoSyncNeeded

	// StateSynced means the Repository was synced (even when no sync was needed it enters this State)
	StateSynced

	// StateError indicates any kind of error, the Repository shouldn't do any more actions
	StateError
)
