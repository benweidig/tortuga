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

// WriteRepositoryStatus writes the current status to the provided Writer
func WriteRepositoryStatus(w io.Writer, repos []*repo.Repository, incomingOnly bool) {
	columnizer := newColumnizer()
	columnizer.AddRow(gchalk.Blue("REPOSITORY"), gchalk.Blue("BRANCH"), gchalk.Blue("STATUS"))

	for _, r := range repos {
		var name string
		var branch string
		var status string

		if r.NeedsSync() {
			name = chalkWhite.Bold(r.Name)
			branch = chalkWhite.Bold(r.Branch)
		} else {
			name = gchalk.Gray(r.Name)
			branch = gchalk.Gray(r.Branch)
		}
		switch r.State {

		case repo.StateRemoteFetched:
			var statusParts []string

			hasIncOut := false
			if r.Incoming > 0 {
				statusParts = append(statusParts, chalkYellowBold.Sprintf("%d↓", r.Incoming))
				hasIncOut = true
			}
			if r.Outgoing > 0 {
				statusParts = append(statusParts, chalkYellowBold.Sprintf("%d↑", r.Outgoing))
				hasIncOut = true
			}

			var changesChalk *gchalk.Builder
			if hasIncOut {
				changesChalk = gchalk.WithWhite()
			} else {
				changesChalk = gchalk.WithGray()
			}

			if r.Changes > 0 {
				statusParts = append(statusParts, changesChalk.Sprintf("%d*", r.Changes))
			} else {
				if r.Noop() {
					statusParts = append(statusParts, gchalk.Gray("-"))
				}
			}

			if r.Unversioned > 0 {
				if r.Incoming > 0 || r.Outgoing > 0 {
					statusParts = append(statusParts, chalkWhite.Sprintf("%d?", r.Unversioned))
				} else {
					statusParts = append(statusParts, chalkGray.Sprintf("%d?", r.Unversioned))
				}

			}

			status = strings.Join(statusParts, " ")

		case repo.StateSynced:
			var statusParts []string

			hasSynced := false

			if r.Incoming > 0 {
				statusParts = append(statusParts, chalkGreenBold.Sprintf("%d↓", r.Incoming))
				hasSynced = true
			}
			if r.Outgoing > 0 {
				if incomingOnly {
					statusParts = append(statusParts, chalkYellow.Sprintf("%d↑", r.Outgoing))
				} else {
					statusParts = append(statusParts, chalkGreenBold.Sprintf("%d↑", r.Outgoing))
					hasSynced = true
				}
			}

			if hasSynced && gchalk.GetLevel() == gchalk.LevelNone {
				statusParts = append(statusParts, "(synced)")
			}

			status = strings.Join(statusParts, " ")

		case repo.StateError:
			name = gchalk.Red(r.Name)
			branch = gchalk.Red(r.Branch)
			status = gchalk.Red(r.Error.Error())

		default:
			status = gchalk.Gray("...")
		}

		columnizer.AddRow(name, branch, status)
	}

	fmt.Fprintln(w, columnizer)
}
