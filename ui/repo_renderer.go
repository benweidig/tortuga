package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/benweidig/tortuga/repo"

	"github.com/jwalton/gchalk"
)

var (
	chalkYellow     = gchalk.WithYellow()
	chalkYellowBold = gchalk.WithYellow().WithBold()
	chalkGreenBold  = gchalk.WithGreen().WithBold()
	chalkWhite      = gchalk.WithWhite()
	chalkGray       = gchalk.WithGray()
)

// repositoryDisplayData contains formatted display information for a repository
type repositoryDisplayData struct {
	name   string
	branch string
	status string
}

// WriteRepositoryStatus writes the current status to the provided Writer
func WriteRepositoryStatus(w io.Writer, repos []*repo.Repository, incomingOnly bool) {
	columnizer := newColumnizer()
	columnizer.AddRow(gchalk.Blue("REPOSITORY"), gchalk.Blue("BRANCH"), gchalk.Blue("STATUS"))

	for _, r := range repos {
		display := formatRepositoryDisplay(r, incomingOnly)
		columnizer.AddRow(display.name, display.branch, display.status)
	}

	fmt.Fprintln(w, columnizer)
}

// formatRepositoryDisplay creates formatted display data for a repository
func formatRepositoryDisplay(r *repo.Repository, incomingOnly bool) repositoryDisplayData {
	display := repositoryDisplayData{
		name:   formatRepositoryName(r),
		branch: formatBranchName(r),
		status: formatRepositoryStatus(r, incomingOnly),
	}
	return display
}

// formatRepositoryName formats the repository name based on sync status
func formatRepositoryName(r *repo.Repository) string {
	if r.State == repo.StateError {
		return gchalk.Red(r.Name)
	}
	if r.NeedsSync() {
		return chalkWhite.Bold(r.Name)
	}
	return gchalk.Gray(r.Name)
}

// formatBranchName formats the branch name based on sync status
func formatBranchName(r *repo.Repository) string {
	if r.State == repo.StateError {
		return gchalk.Red(r.Branch)
	}
	if r.NeedsSync() {
		return chalkWhite.Bold(r.Branch)
	}
	return gchalk.Gray(r.Branch)
}

// formatRepositoryStatus formats the status column based on repository state
func formatRepositoryStatus(r *repo.Repository, incomingOnly bool) string {
	switch r.State {
	case repo.StateRemoteFetched:
		return formatRemoteFetchedStatus(r)
	case repo.StateSynced:
		return formatSyncedStatus(r, incomingOnly)
	case repo.StateError:
		return gchalk.Red(r.Error.Error())
	default:
		return gchalk.Gray("...")
	}
}

// formatRemoteFetchedStatus formats status for repositories that have been fetched
func formatRemoteFetchedStatus(r *repo.Repository) string {
	var parts []string

	// Add incoming/outgoing indicators
	hasIncOut := false
	if r.Incoming > 0 {
		parts = append(parts, chalkYellowBold.Sprintf("%d↓", r.Incoming))
		hasIncOut = true
	}
	if r.Outgoing > 0 {
		parts = append(parts, chalkYellowBold.Sprintf("%d↑", r.Outgoing))
		hasIncOut = true
	}

	// Add local changes indicator
	if r.Changes > 0 {
		color := chalkGray
		if hasIncOut {
			color = chalkWhite
		}
		parts = append(parts, color.Sprintf("%d*", r.Changes))
	} else if r.Noop() {
		parts = append(parts, gchalk.Gray("-"))
	}

	// Add unversioned files indicator
	if r.Unversioned > 0 {
		color := chalkGray
		if hasIncOut {
			color = chalkWhite
		}
		parts = append(parts, color.Sprintf("%d?", r.Unversioned))
	}

	return strings.Join(parts, " ")
}

// formatSyncedStatus formats status for repositories that have been synced
func formatSyncedStatus(r *repo.Repository, incomingOnly bool) string {
	var parts []string
	hasSynced := false

	if r.Incoming > 0 {
		parts = append(parts, chalkGreenBold.Sprintf("%d↓", r.Incoming))
		hasSynced = true
	}

	if r.Outgoing > 0 {
		if incomingOnly {
			parts = append(parts, chalkYellow.Sprintf("%d↑", r.Outgoing))
		} else {
			parts = append(parts, chalkGreenBold.Sprintf("%d↑", r.Outgoing))
			hasSynced = true
		}
	}

	// Add "(synced)" indicator when colors are disabled
	if hasSynced && gchalk.GetLevel() == gchalk.LevelNone {
		parts = append(parts, "(synced)")
	}

	return strings.Join(parts, " ")
}
