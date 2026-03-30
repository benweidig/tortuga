package ui

import (
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/muesli/termenv"
	"golang.org/x/term"

	"github.com/benweidig/tortuga/internal/model"
)

// Renderer is a synchronous, in-place terminal renderer. It owns no goroutines
// or channels — the caller controls when to render. Call Render whenever the
// state changes; it resets the previous output and redraws from scratch.
type Renderer struct {
	writer        *StdoutWriter
	output        *termenv.Output
	hasContent    bool
	colWidths     []int // remembered from last render, applied as minimums
	lastHeight    int
	lastWidth     int
	tableRowCount int // rows in the last rendered table (header + visible repos [+ "more" line])
}

func NewRenderer(writer *StdoutWriter, monochrome bool) *Renderer {
	profile := termenv.ColorProfile()
	if monochrome {
		profile = termenv.Ascii
	}
	// NewOutput targets the writer so all styled strings flow through the
	// StdoutWriter buffer and are flushed together.
	output := termenv.NewOutput(writer, termenv.WithProfile(profile))
	return &Renderer{writer: writer, output: output}
}

// Render resets the previous output (if any) and draws the current table.
// spinnerFrame is the active braille frame, or "" when fetch is complete.
func (r *Renderer) Render(repos []model.Repo, spinnerFrame string) {
	if r.hasContent {
		r.writer.Reset()
	}

	width, height, _ := term.GetSize(int(os.Stdout.Fd()))
	if height == 0 {
		height = 24
	}
	r.lastHeight = height
	r.lastWidth = width

	r.renderTable(repos, spinnerFrame, height)
	r.writer.Flush()
	r.hasContent = true
}

// RenderFull resets the previous output and draws the complete table without
// height-based truncation.
func (r *Renderer) RenderFull(repos []model.Repo) {
	if r.hasContent {
		r.writer.Reset()
	}
	r.renderTable(repos, "", math.MaxInt)
	r.writer.Flush()
	r.hasContent = true
}

// StartFresh marks the renderer as having no prior content so the next Render
// call prints below the current cursor position instead of resetting to the
// top of the previous output. It also resets the writer's line counter so
// in-phase resets during the next phase are relative to that phase's own
// first render.
func (r *Renderer) StartFresh() {
	r.hasContent = false
	r.writer.ResetLineCount()
}

// RenderErrors prints the error detail block below the table. It is called
// once after a phase completes, not on every tick.
func (r *Renderer) RenderErrors(repos []model.Repo) {
	var failed []model.Repo
	for _, repo := range repos {
		if repo.Err != nil {
			failed = append(failed, repo)
		}
	}
	if len(failed) == 0 {
		return
	}

	// Compute name column width to align the error block with the table.
	nameWidth := 0
	for _, repo := range failed {
		if len(repo.Name) > nameWidth {
			nameWidth = len(repo.Name)
		}
	}

	fmt.Fprintln(r.writer, "\nErrors:")
	for _, repo := range failed {
		name := fmt.Sprintf("%-*s", nameWidth, repo.Name)
		fmt.Fprintf(r.writer, "  %s  %s\n", name, repo.Err.Message)
	}
	r.writer.Flush()
}

// renderTable writes the full table to the writer.
func (r *Renderer) renderTable(repos []model.Repo, spinnerFrame string, termHeight int) {
	table := newColumnizer()
	table.setMinWidths(r.colWidths)
	table.AddRow(
		r.output.String("REPOSITORY").Foreground(termenv.ANSIBlue).Bold().String(),
		r.output.String("BRANCH").Foreground(termenv.ANSIBlue).Bold().String(),
		r.output.String("STATUS").Foreground(termenv.ANSIBlue).Bold().String(),
	)

	// Reserve rows for: header + potential "... and N more" line.
	const reservedRows = 2
	maxRepoRows := max(termHeight-reservedRows, 1)

	visible := repos
	truncated := 0
	if len(repos) > maxRepoRows {
		visible = repos[:maxRepoRows-1]
		truncated = len(repos) - (maxRepoRows - 1)
	}

	r.tableRowCount = 1 + len(visible) // header + visible repos
	if truncated > 0 {
		r.tableRowCount++ // "... and N more" line
	}

	for _, repo := range visible {
		table.AddRow(
			r.nameCell(repo),
			r.branchCell(repo),
			r.statusCell(repo, spinnerFrame),
		)
	}

	r.colWidths = table.columnWidths()
	fmt.Fprint(r.writer, table.String())

	if truncated > 0 {
		dim := r.output.String(fmt.Sprintf("... and %d more", truncated)).Faint().String()
		fmt.Fprintln(r.writer, dim)
	}
}

// visibleCount returns how many repo rows fit in the table at the given height.
func (r *Renderer) visibleCount(repos []model.Repo, termHeight int) int {
	const reservedRows = 2
	maxRepoRows := termHeight - reservedRows
	if maxRepoRows < 1 {
		maxRepoRows = 1
	}
	if len(repos) <= maxRepoRows {
		return len(repos)
	}
	return maxRepoRows - 1
}

// renderSingleRow formats one repo row without a trailing newline, using the
// stored column widths so partial updates stay aligned with the full table.
func (r *Renderer) renderSingleRow(repo model.Repo, spinnerFrame string) string {
	row := newColumnizer()
	row.setMinWidths(r.colWidths)
	row.AddRow(r.nameCell(repo), r.branchCell(repo), r.statusCell(repo, spinnerFrame))
	return strings.TrimRight(row.String(), "\n")
}

// isSpinning reports whether a repo's status cell includes the spinner frame.
func isSpinning(repo model.Repo) bool {
	switch repo.WorkStatus {
	case model.WorkStashing, model.WorkPulling, model.WorkPushing, model.WorkUnstashing:
		return true
	}
	return repo.FetchStatus == model.FetchRunning
}

// RenderTick updates only the rows whose status cell contains the spinner.
// Falls back to a full Render on terminal resize.
func (r *Renderer) RenderTick(repos []model.Repo, spinnerFrame string) {
	if !r.hasContent {
		return
	}
	width, height, _ := term.GetSize(int(os.Stdout.Fd()))
	if height == 0 {
		height = 24
	}
	if height != r.lastHeight || width != r.lastWidth {
		r.Render(repos, spinnerFrame)
		return
	}
	count := r.visibleCount(repos, height)
	for i := 0; i < count; i++ {
		if isSpinning(repos[i]) {
			rowsFromBottom := r.tableRowCount - (i + 1)
			r.writer.UpdateLine(rowsFromBottom, r.renderSingleRow(repos[i], spinnerFrame))
		}
	}
}

// RenderRowUpdate updates a single repo row in place.
// Falls back to a full Render on terminal resize or if the repo is truncated.
func (r *Renderer) RenderRowUpdate(idx int, repos []model.Repo, spinnerFrame string) {
	if !r.hasContent {
		return
	}
	width, height, _ := term.GetSize(int(os.Stdout.Fd()))
	if height == 0 {
		height = 24
	}
	if height != r.lastHeight || width != r.lastWidth {
		r.Render(repos, spinnerFrame)
		return
	}
	if idx >= r.visibleCount(repos, height) {
		// Repo is not visible (truncated) — a full redraw is needed.
		r.Render(repos, spinnerFrame)
		return
	}
	rowsFromBottom := r.tableRowCount - (idx + 1)
	r.writer.UpdateLine(rowsFromBottom, r.renderSingleRow(repos[idx], spinnerFrame))
}

func (r *Renderer) nameCell(repo model.Repo) string {
	if repo.NoUpstream {
		return r.output.String(repo.Name).Foreground(termenv.ANSIYellow).String()
	}
	if repo.Ahead > 0 || repo.Behind > 0 {
		return r.output.String(repo.Name).Bold().String()
	}
	return r.output.String(repo.Name).Faint().String()
}

func (r *Renderer) branchCell(repo model.Repo) string {
	if repo.NoUpstream {
		return r.output.String(repo.Branch).Foreground(termenv.ANSIYellow).String()
	}
	if repo.Ahead > 0 || repo.Behind > 0 {
		return r.output.String(repo.Branch).Bold().String()
	}
	return r.output.String(repo.Branch).Faint().String()
}

func (r *Renderer) statusCell(repo model.Repo, spinnerFrame string) string {
	// Phase 2 status takes precedence once work has started.
	if repo.WorkStatus != model.WorkPending {
		return r.workStatusCell(repo, spinnerFrame)
	}
	return r.fetchStatusCell(repo, spinnerFrame)
}

func (r *Renderer) fetchStatusCell(repo model.Repo, spinnerFrame string) string {
	switch repo.FetchStatus {
	case model.FetchPending:
		return r.output.String("–").Faint().String()

	case model.FetchRunning:
		if spinnerFrame == "" {
			return r.output.String("–").Faint().String()
		}
		return spinnerFrame

	case model.FetchError:
		msg := "error"
		if repo.Err != nil {
			msg = repo.Err.Message
		}
		return r.output.String(msg).Foreground(termenv.ANSIRed).String()

	case model.FetchDone:
		if repo.Err != nil {
			return r.output.String(repo.Err.Message).Foreground(termenv.ANSIRed).String()
		}
		if repo.NoUpstream {
			return r.output.String("no upstream").Foreground(termenv.ANSIYellow).String()
		}
		return r.buildDoneStatus(repo)
	}
	return ""
}

func (r *Renderer) workStatusCell(repo model.Repo, spinnerFrame string) string {
	spin := spinnerFrame
	if spin == "" {
		spin = "–"
	}

	switch repo.WorkStatus {
	case model.WorkDone:
		return r.output.String("✓").Foreground(termenv.ANSIGreen).String()
	case model.WorkError:
		return r.output.String("error").Foreground(termenv.ANSIRed).String()
	default:
		return spin
	}
}

func (r *Renderer) buildDoneStatus(repo model.Repo) string {
	actionable := repo.Ahead > 0 || repo.Behind > 0

	var parts []string

	if repo.Ahead > 0 {
		parts = append(parts, fmt.Sprintf("%d↑", repo.Ahead))
	}

	if repo.Behind > 0 {
		parts = append(parts, fmt.Sprintf("%d↓", repo.Behind))
	}

	if repo.Status.ChangedFiles > 0 {
		parts = append(parts, fmt.Sprintf("%d*", repo.Status.ChangedFiles))
	}

	if repo.Status.UnversionedFiles > 0 {
		parts = append(parts, fmt.Sprintf("%d?", repo.Status.UnversionedFiles))
	}

	if len(parts) == 0 {
		return r.output.String("✓").Foreground(termenv.ANSIGreen).String()
	}

	if !actionable {
		return r.output.String(strings.Join(parts, " ")).Faint().String()
	}

	// For actionable repos, colour ahead/behind yellow and dirty/unversioned faint.
	var styled []string

	if repo.Ahead > 0 {
		styled = append(styled, r.output.String(fmt.Sprintf("%d↑", repo.Ahead)).Foreground(termenv.ANSIYellow).Bold().String())
	}

	if repo.Behind > 0 {
		styled = append(styled, r.output.String(fmt.Sprintf("%d↓", repo.Behind)).Foreground(termenv.ANSIYellow).Bold().String())
	}

	if repo.Status.ChangedFiles > 0 {
		styled = append(styled, fmt.Sprintf("%d*", repo.Status.ChangedFiles))
	}

	if repo.Status.UnversionedFiles > 0 {
		styled = append(styled, fmt.Sprintf("%d?", repo.Status.UnversionedFiles))
	}

	return strings.Join(styled, " ")
}
