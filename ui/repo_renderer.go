package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/benweidig/tortuga/repo"

	"github.com/jwalton/gchalk"
)

// WriteRepositoryStatus writes the current status to the provided Writer
func WriteRepositoryStatus(w io.Writer, repos []*repo.Repository, incomingOnly bool) {
	columnizer := newColumnizer()
	columnizer.AddRow(gchalk.Blue("REPOSITORY"), gchalk.Blue("BRANCH"), gchalk.Blue("STATUS"))

	for _, r := range repos {
		var name string
		var branch string
		var status string
		switch r.State {

		case repo.StateRemoteFetched:
			var statusParts []string

			if r.Incoming > 0 {
				statusParts = append(statusParts, gchalk.WithBrightYellow().WithBold().Sprintf("%d↓", r.Incoming))
			}
			if r.Outgoing > 0 {
				statusParts = append(statusParts, gchalk.WithBrightYellow().WithBold().Sprintf("%d↑", r.Outgoing))
			}

			if r.Changes > 0 {
				statusParts = append(statusParts, gchalk.WithYellow().Sprintf("%d*", r.Changes))
			} else {
				statusParts = append(statusParts, gchalk.Green("0*"))
			}

			if r.Unversioned > 0 {
				statusParts = append(statusParts, gchalk.WithYellow().Sprintf("%d?", r.Unversioned))
			}

			if r.NeedsSync() {
				name = gchalk.WithWhite().Bold(r.Name)
				branch = gchalk.WithWhite().Bold(r.Branch)
			} else {
				name = gchalk.Gray(r.Name)
				branch = gchalk.Gray(r.Branch)
			}

			status = strings.Join(statusParts, " ")

		case repo.StateSynced:
			var statusParts []string

			g := gchalk.WithWhite()

			if r.Incoming > 0 || (!incomingOnly && r.Outgoing > 0) {
				g = g.WithBold()
			}

			if r.Incoming > 0 {
				statusParts = append(statusParts, gchalk.WithGreen().WithBold().Sprintf("%d↓", r.Incoming))
			}
			if r.Outgoing > 0 {
				if incomingOnly {
					statusParts = append(statusParts, gchalk.WithYellow().Sprintf("%d↑", r.Outgoing))
				} else {
					statusParts = append(statusParts, gchalk.WithGreen().WithBold().Sprintf("%d↑", r.Outgoing))
				}
			}

			name = g.White(r.Name)
			branch = g.White(r.Branch)
			status = strings.Join(statusParts, " ")

		case repo.StateError:
			name = gchalk.Red(r.Name)
			branch = gchalk.Red(r.Branch)
			status = gchalk.Red(r.Error.Error())

		default:
			name = gchalk.Gray(r.Name)
			branch = gchalk.Gray(r.Branch)
			status = gchalk.Gray("...")
		}

		columnizer.AddRow(name, branch, status)
	}

	fmt.Fprintln(w, columnizer)
}
