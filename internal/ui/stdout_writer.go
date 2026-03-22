package ui

import (
	"bytes"
	"fmt"
	"os"

	"github.com/muesli/termenv"
)

// StdoutWriter is an in-place writer for stdout that tracks cursor position
// so it can move back up and redraw content in-place.
// Note: designed for single-threaded use.
type StdoutWriter struct {
	buffer     bytes.Buffer
	direct     *termenv.Output
	lineBreaks int
}

func NewStdoutWriter() *StdoutWriter {
	return &StdoutWriter{
		direct: termenv.NewOutput(os.Stdout),
	}
}

// Write adds to the internal buffer. Content is not written to stdout until
// Flush is called.
func (w *StdoutWriter) Write(b []byte) (n int, err error) {
	return w.buffer.Write(b)
}

// Reset moves the cursor back to the very beginning of all output written so
// far and clears it, then resets all internal counters.
func (w *StdoutWriter) Reset() {
	w.reset(w.lineBreaks)
	w.buffer.Reset()
	w.lineBreaks = 0
}

// ResetLineCount zeroes the internal line counter without moving the cursor.
// Use this when subsequent resets should be relative to the current cursor
// position rather than the start of all previous output.
func (w *StdoutWriter) ResetLineCount() {
	w.lineBreaks = 0
}

func (w *StdoutWriter) reset(lineBreaks int) {
	w.direct.ClearLines(lineBreaks)
}

// UpdateLine overwrites a single already-rendered line in place.
// rowsFromBottom is the number of rows from the current cursor (which sits one
// row below the last rendered line) up to the target row. Does not affect the
// line-break counter since no lines are added or removed.
func (w *StdoutWriter) UpdateLine(rowsFromBottom int, content string) {
	if rowsFromBottom > 0 {
		w.direct.CursorUp(rowsFromBottom)
	}
	fmt.Fprint(os.Stdout, "\r")
	fmt.Fprint(os.Stdout, content)
	w.direct.ClearLineRight()
	if rowsFromBottom > 0 {
		fmt.Fprint(os.Stdout, "\r")
		w.direct.CursorDown(rowsFromBottom)
	}
}

// Flush writes the buffered content to stdout and updates the line-break
// counter. The buffer is cleared after writing.
func (w *StdoutWriter) Flush() error {
	bufferBytes := w.buffer.Bytes()
	if len(bufferBytes) == 0 {
		return nil
	}
	w.lineBreaks += bytes.Count(bufferBytes, []byte("\n"))
	_, err := os.Stdout.Write(bufferBytes)
	w.buffer.Reset()
	return err
}
