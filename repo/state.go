package repo

type State int

const (
	StateNone State = iota
	StateUpdated
	StateSynced

	StateAuthError
	StateError
)
