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
	c := Changes{
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
			c.Modified++
		case strings.HasPrefix(row, " A "):
			c.Added++
		case strings.HasPrefix(row, " D "):
			c.Deleted++
		case strings.HasPrefix(row, " R "):
			c.Renamed++
		case strings.HasPrefix(row, " C "):
			c.Copied++
		case strings.HasPrefix(row, " U "):
			c.UpdatedUnmerged++
		case strings.HasPrefix(row, "?? "):
			c.Unversioned++
		}
	}

	c.Stashable = c.Modified + c.Added + c.Deleted + c.Renamed + c.Copied + c.UpdatedUnmerged
	c.Total = c.Stashable + c.Unversioned

	return c
}
