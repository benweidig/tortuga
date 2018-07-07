// +build !windows

package ui

import (
	"fmt"
	"os"
)

func (w *StdoutWriter) reset() {
	for i := 0; i < w.lineCount; i++ {
		fmt.Fprintf(os.Stdout, "%c[2K", ESCAPE)     // clear the line
		fmt.Fprintf(os.Stdout, "%c[%dA", ESCAPE, 1) // move the cursor up
	}
}
