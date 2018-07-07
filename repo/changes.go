package repo

import (
	"bufio"
	"bytes"
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
		if len(row) < 3 {
			continue
		}
		prefix := string(row[0:3])
		switch prefix {
		case " M ":
			c.Modified++
		case " A ":
			c.Added++
		case " D ":
			c.Deleted++
		case " R ":
			c.Renamed++
		case " C ":
			c.Copied++
		case " U ":
			c.UpdatedUnmerged++
		case "?? ":
			c.Unversioned++
		}
	}

	c.Stashable = c.Modified + c.Added + c.Deleted + c.Renamed + c.Copied + c.UpdatedUnmerged
	c.Total = c.Stashable + c.Unversioned

	return c
}
