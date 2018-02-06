package repo

// A Repository State representing the current state in the Tortuga workflow
type State int

const (
	// Initial state for new Repositories
	StateNone State = iota

	// The Repository was updated and has local changes and Incoming/Outgoung count
	StateUpdated

	// All changes have been synced (even when no sync was needed it enters this State)
	StateSynced

	// A command failed, the repository shouldn't do any more actions
	StateError
)
