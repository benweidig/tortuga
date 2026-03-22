// Package ui handles all terminal output: the live table renderer, the
// interactive sync prompt, and the spinner ticker that drives partial redraws.
package ui

import (
	"regexp"
	"strings"

	"github.com/mattn/go-runewidth"
)

// columnizer builds a left-aligned, pipe-separated table whose column widths
// are computed from the content. ANSI escape sequences are stripped before
// measuring so styled text aligns correctly.
type columnizer struct {
	rows      []*columnizerRow
	minWidths []int
}

func (t *columnizer) setMinWidths(widths []int) {
	t.minWidths = widths
}

func newColumnizer() *columnizer {
	return &columnizer{}
}

func (t *columnizer) AddRow(contents ...string) {
	row := newColumnizerRow(contents...)
	t.rows = append(t.rows, row)
}

// columnWidths returns the effective width of each column, incorporating any
// minimums set via setMinWidths.
func (t *columnizer) columnWidths() []int {
	var widths []int
	for _, row := range t.rows {
		for i, cell := range row.cells {
			if i+1 > len(widths) {
				widths = append(widths, 0)
			}
			if cell.displayWidth > widths[i] {
				widths[i] = cell.displayWidth
			}
		}
	}
	for i := range min(len(t.minWidths), len(widths)) {
		if t.minWidths[i] > widths[i] {
			widths[i] = t.minWidths[i]
		}
	}
	return widths
}

// String returns string representation of the table
func (t *columnizer) String() string {

	// Empty table == empty string
	if len(t.rows) == 0 {
		return ""
	}

	colWidths := t.columnWidths()

	// All columns except the last get a trailing " │ " separator.
	cols := len(colWidths)
	borderedCols := cols - 1

	var builder strings.Builder

	for _, row := range t.rows {
		for colIdx := range cols {
			colWidth := colWidths[colIdx]

			// Rows don't need to have the same amount of cells so we might need to fill up
			// the empty cells with spaces
			if colIdx < len(row.cells) {
				cell := row.cells[colIdx]
				builder.WriteString(cell.paddedContent(colWidth))
			} else {
				if colIdx < cols-1 {
					for range colWidth {
						builder.WriteByte(' ')
					}
				}
			}

			if colIdx < borderedCols {
				builder.WriteString(" │ ")
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

var ansiColorCodesRegexp = regexp.MustCompile(`\x1b\[[0-9;]*m`)

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
	for range colWidth - c.displayWidth {
		builder.WriteByte(' ')
	}
	return builder.String()
}
