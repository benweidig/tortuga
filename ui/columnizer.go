package ui

import (
	"regexp"
	"strings"
	"sync"

	"github.com/mattn/go-runewidth"
)

type columnizer struct {
	rows []*columnizerRow
	mtx  *sync.RWMutex
}

// NewColumnizer creates a new table with sensible defaults
func newColumnizer() *columnizer {
	return &columnizer{
		mtx: new(sync.RWMutex),
	}
}

func (t *columnizer) AddRow(contents ...string) {
	// We don't want to have a half-build table so we need a lock for updating content
	t.mtx.Lock()
	defer t.mtx.Unlock()

	row := newColumnizerRow(contents...)
	t.rows = append(t.rows, row)
}

// Returns string representation of the table
func (t *columnizer) String() string {
	// We want to make sure the data won't change mid string building
	t.mtx.RLock()
	defer t.mtx.RUnlock()

	// Empty table == empty string
	if len(t.rows) == 0 {
		return ""
	}

	// Determinate the width of each column
	var colWidths []int
	for _, row := range t.rows {
		for i, cell := range row.cells {
			if i+1 > len(colWidths) {
				colWidths = append(colWidths, 0)
			}

			if cell.displayWidth > colWidths[i] {
				colWidths[i] = cell.displayWidth
			}
		}
	}

	// Remove outer border
	cols := len(colWidths)
	borderedCols := cols
	borderedCols--

	// Holds the string representation of the table
	var builder strings.Builder

	// Build table data
	for _, row := range t.rows {
		for colIdx := 0; colIdx < cols; colIdx++ {
			colWidth := colWidths[colIdx]

			// Rows don't need to have the same amount of cells so we might need to fill up
			// the empty cells with spaces
			if colIdx < len(row.cells) {
				cell := row.cells[colIdx]
				builder.WriteString(cell.paddedContent(colWidth))
			} else {
				if colIdx < cols-1 {
					for i := 0; i < colWidth; i++ {
						builder.WriteByte(' ')
					}
				}
			}

			if colIdx < borderedCols {
				builder.WriteString(" | ")
			}
		}
		builder.WriteString("\n")
	}

	return builder.String()
}

type columnizerRow struct {
	cells []*columnizerCell
}

func newColumnizerRow(contents ...string) *columnizerRow {
	r := &columnizerRow{
		cells: make([]*columnizerCell, len(contents)),
	}

	for idx, content := range contents {
		cell := newColumnizerCell(content)
		r.cells[idx] = cell
	}

	return r
}

type columnizerCell struct {
	content      string
	displayWidth int
}

var ansiColorCodesRegexp = regexp.MustCompile("\\x1b\\[[0-9;]*m")

func newColumnizerCell(content string) *columnizerCell {
	// We need to remove ANSI color codes to get the actual width
	sanitized := ansiColorCodesRegexp.ReplaceAllString(content, "")

	return &columnizerCell{
		content:      content,
		displayWidth: runewidth.StringWidth(sanitized),
	}
}

func (c *columnizerCell) paddedContent(colWidth int) string {
	var builder strings.Builder
	builder.Grow(colWidth)
	builder.WriteString(c.content)
	for i := 0; i < colWidth-c.displayWidth; i++ {
		builder.WriteByte(' ')
	}
	return builder.String()
}
