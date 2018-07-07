package cmd

import (
	"fmt"
	"io"
	"strings"

	"github.com/benweidig/tortuga/git"
	"github.com/benweidig/tortuga/repo"
	"github.com/benweidig/tortuga/ui"

	"github.com/fatih/color"
)

func writeCurrentStatus(w io.Writer, repos []*repo.Repository) {
	columnizer := ui.NewColumnizer()
	columnizer.AddRow("REPOSITORY", "BRANCH", "STATUS")

	for _, r := range repos {
		var status string
		switch r.State {
		case repo.StateChangesUpdated:
			var statusParts []string
			if r.LocalChanges.Total == 0 {
				statusParts = append(statusParts, color.GreenString("%d*", r.LocalChanges.Total))
			} else {
				statusParts = append(statusParts, color.YellowString("%d*", r.LocalChanges.Total))
			}

			if r.Incoming > 0 {
				statusParts = append(statusParts, color.YellowString("%d↓", r.Incoming))
			}
			if r.Outgoing > 0 {
				statusParts = append(statusParts, color.YellowString("%d↑", r.Outgoing))
			}
			status = strings.Join(statusParts, " ")

		case repo.StateError:
			switch r.Error {
			case git.ErrorAuthentication:
				status = color.RedString("Auth Error")
			case git.ErrorNoUpstream:
				status = color.RedString("No upstream")
			default:
				status = color.RedString("Error")
			}
		default:
			status = "..."
		}
		columnizer.AddRow(color.WhiteString(r.Name), color.WhiteString(r.Branch), status)
	}

	fmt.Fprintln(w, columnizer)
}

func writeActionsTaken(w io.Writer, repos []*repo.Repository) {
	columnizer := ui.NewColumnizer()
	columnizer.AddRow("PROJECT", "BRANCH", "ACTIONS")

	for _, r := range repos {
		var status string
		switch r.State {
		case repo.StateSynced:
			if r.Outgoing == 0 && r.Incoming == 0 {
				status = "Nothing to do"
			} else {
				var statusParts []string
				if r.Incoming > 0 {
					statusParts = append(statusParts, fmt.Sprintf("%d↓", r.Incoming))
				}
				if r.Outgoing > 0 {
					statusParts = append(statusParts, fmt.Sprintf("%d↑", r.Outgoing))
				}
				status = strings.Join(statusParts, ", ")
			}
			status = color.GreenString(status)

		case repo.StateError:
			status = color.RedString("Error")
		default:
			status = "..."
		}
		columnizer.AddRow(color.New(color.FgWhite).Sprint(r.Name), color.New(color.FgWhite).Sprint(r.Branch), color.New(color.FgWhite).Sprint(status))
	}

	fmt.Fprintln(w, columnizer)
}
