package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/benweidig/tortuga/repo"

	"github.com/fatih/color"
)

// WriteRepositoryStatus writes the current status to the provided Writer
func WriteRepositoryStatus(w io.Writer, repos []*repo.Repository) {
	columnizer := newColumnizer()
	columnizer.AddRow("REPOSITORY", "BRANCH", "STATUS")

	for _, r := range repos {
		var status string
		switch r.State {

		case repo.StateRemoteFetched:
			var statusParts []string

			if r.Incoming > 0 {
				statusParts = append(statusParts, color.YellowString("%d↓", r.Incoming))
			}
			if r.Outgoing > 0 {
				statusParts = append(statusParts, color.YellowString("%d↑", r.Outgoing))
			}

			if r.Changes > 0 {
				statusParts = append(statusParts, color.YellowString("%d*", r.Changes))
			} else {
				statusParts = append(statusParts, color.GreenString("0*"))
			}

			if r.Unversioned > 0 {
				statusParts = append(statusParts, color.YellowString("%d?", r.Unversioned))
			}

			status = strings.Join(statusParts, " ")

		case repo.StateSynced:
			if r.Outgoing == 0 && r.Incoming == 0 {
				status = color.GreenString("Nothing to do")
			} else {
				var statusParts []string
				if r.Incoming > 0 {
					statusParts = append(statusParts, color.GreenString("%d↓", r.Incoming))
				}
				if r.Outgoing > 0 {
					statusParts = append(statusParts, color.GreenString("%d↑", r.Outgoing))
				}
				status = strings.Join(statusParts, " ")
			}

		case repo.StateError:
			status = color.RedString(r.Error.Error())
		default:
			status = "..."
		}
		columnizer.AddRow(color.WhiteString(r.Name), color.WhiteString(r.Branch), status)
	}

	fmt.Fprintln(w, columnizer)
}
