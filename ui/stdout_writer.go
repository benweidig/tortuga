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
	buffer    bytes.Buffer
	mtx       *sync.Mutex
	lineCount int
}

// NewStdoutWriter returns a new Writer
func NewStdoutWriter() *StdoutWriter {
	return &StdoutWriter{
		mtx: &sync.Mutex{},
	}
}

// Flush writes to os.Stdout out and resets the cursor position and buffer.
func (w *StdoutWriter) Flush() error {
	w.mtx.Lock()
	defer w.mtx.Unlock()

	bufferBytes := w.buffer.Bytes()

	// do nothing if buffer is empty
	if len(bufferBytes) == 0 {
		return nil
	}
	w.reset()

	lines := bytes.Count(bufferBytes, []byte("\n"))
	w.lineCount = lines

	_, err := os.Stdout.Write(bufferBytes)
	w.buffer.Reset()
	return err
}

// Write adds to its buffers.
func (w *StdoutWriter) Write(b []byte) (n int, err error) {
	w.mtx.Lock()
	defer w.mtx.Unlock()
	return w.buffer.Write(b)
}
