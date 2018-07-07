package repo

// State represents representing the current state of a Repository in the Tortuga workflow
type State int

const (
	// StateNone is the initial state
	StateNone State = iota

	// StateChangesUpdated means the Repository was fetched from upstream (if not local-mode)
	// and has local changes and Incoming/Outgoung count set
	StateChangesUpdated

	// StateSynced means the Repository was synced (even when no sync was needed it enters this State)
	StateSynced

	// StateError indicates any kind of error, the Repository shouldn't do any more actions
	StateError
)
