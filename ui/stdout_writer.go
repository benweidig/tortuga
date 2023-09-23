package ui

import (
	"bytes"
	"os"
	"sync"
)

// ESCAPE is the ASCII code for escape character
const ESCAPE = 27

// StdoutWriter is an "in-place" writer for the StdOut
type StdoutWriter struct {
	buffer             bytes.Buffer
	writeMtx           *sync.Mutex
	renderMtx          *sync.Mutex
	lineBreaks         int
	preserveLineBreaks int
}

// NewStdoutWriter returns a new Writer
func NewStdoutWriter() *StdoutWriter {
	return &StdoutWriter{
		writeMtx:  &sync.Mutex{},
		renderMtx: &sync.Mutex{},
	}
}

// Write adds to its buffers.
func (w *StdoutWriter) Write(b []byte) (n int, err error) {
	w.writeMtx.Lock()
	defer w.writeMtx.Unlock()

	return w.buffer.Write(b)
}

// Mark a position for partial resets
func (w *StdoutWriter) Mark() {
	w.writeMtx.Lock()
	defer w.writeMtx.Unlock()

	bufferBytes := w.buffer.Bytes()

	// No content mean we need no marker
	if len(bufferBytes) == 0 {
		w.preserveLineBreaks = w.lineBreaks
		return
	}

	// Calculate the lines of the current buffer and
	// mark the last line
	lines := bytes.Count(bufferBytes, []byte("\n"))
	w.preserveLineBreaks = lines
}

// Reset the StdoutWriter to 0
func (w *StdoutWriter) Reset() {
	w.writeMtx.Lock()
	defer w.writeMtx.Unlock()

	w.reset(w.lineBreaks)
	w.buffer.Reset()

	w.lineBreaks = 0
	w.preserveLineBreaks = 0
}

// ResetToMarker reset the StdoutWriter to the marked position
func (w *StdoutWriter) ResetToMarker() {
	w.writeMtx.Lock()
	defer w.writeMtx.Unlock()

	diff := w.lineBreaks - w.preserveLineBreaks
	w.reset(diff)
	w.buffer.Reset()

	w.lineBreaks = w.preserveLineBreaks
}

// AddLineBreaks to react to cursor changes not done via the writer
func (w *StdoutWriter) AddLineBreaks(amount int) {
	w.lineBreaks += amount
}

// Flush writes to os.Stdout out and resets the cursor position and buffer.
func (w *StdoutWriter) Flush() error {
	w.writeMtx.Lock()
	defer w.writeMtx.Unlock()

	bufferBytes := w.buffer.Bytes()

	// do nothing if buffer is empty
	if len(bufferBytes) == 0 {
		return nil
	}

	breaks := bytes.Count(bufferBytes, []byte("\n"))

	w.lineBreaks += breaks

	_, err := os.Stdout.Write(bufferBytes)
	w.buffer.Reset()
	return err
}

// Render is a mutex locked helper to reset, write, and flush
func (w *StdoutWriter) Render(fn func()) {
	w.renderMtx.Lock()
	defer w.renderMtx.Unlock()

	w.Reset()
	fn()
	w.Flush()
}
