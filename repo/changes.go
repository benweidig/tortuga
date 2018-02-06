package repo

import (
	"bufio"
	"bytes"
	"strings"
)

type Changes struct {
	Modified        int
	Added           int
	Deleted         int
	Renamed         int
	Copied          int
	UpdatedUnmerged int
	Unversioned     int
	Total           int
	Stashable       int
}

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
			s.Modified += 1
		case strings.HasPrefix(row, " A "):
			s.Added += 1
		case strings.HasPrefix(row, " D "):
			s.Deleted += 1
		case strings.HasPrefix(row, " R "):
			s.Renamed += 1
		case strings.HasPrefix(row, " C "):
			s.Copied += 1
		case strings.HasPrefix(row, " U "):
			s.UpdatedUnmerged += 1
		case strings.HasPrefix(row, "?? "):
			s.Unversioned += 1
		}
	}

	s.Stashable = s.Modified + s.Added + s.Deleted + s.Renamed + s.Copied + s.UpdatedUnmerged
	s.Total = s.Stashable + s.Unversioned

	return s
}
