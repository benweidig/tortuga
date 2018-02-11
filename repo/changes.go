package repo

import (
	"bufio"
	"bytes"
	"strings"
)

// Changes represents the differences between the index file and the current HEAD commit
type Changes struct {
	Modified        int
	Added           int
	Deleted         int
	Renamed         int
	Copied          int
	UpdatedUnmerged int
	Unversioned     int
	Stashable       int
	Total           int
}

// NewChanges creates a new Changes struct based on the stdOut of git status --porcelain
func NewChanges(stdOut bytes.Buffer) Changes {
	s := Changes{
		Modified:        0,
		Added:           0,
		Deleted:         0,
		Renamed:         0,
		Copied:          0,
		UpdatedUnmerged: 0,
		Unversioned:     0,
	}
	scanner := bufio.NewScanner(&stdOut)

	for scanner.Scan() {
		row := scanner.Text()

		switch {
		case strings.HasPrefix(row, " M "):
			s.Modified++
		case strings.HasPrefix(row, " A "):
			s.Added++
		case strings.HasPrefix(row, " D "):
			s.Deleted++
		case strings.HasPrefix(row, " R "):
			s.Renamed++
		case strings.HasPrefix(row, " C "):
			s.Copied++
		case strings.HasPrefix(row, " U "):
			s.UpdatedUnmerged++
		case strings.HasPrefix(row, "?? "):
			s.Unversioned++
		}
	}

	s.Stashable = s.Modified + s.Added + s.Deleted + s.Renamed + s.Copied + s.UpdatedUnmerged
	s.Total = s.Stashable + s.Unversioned

	return s
}
