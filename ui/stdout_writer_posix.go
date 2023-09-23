//go:build !windows
// +build !windows

package ui

import (
	"fmt"
	"os"
)

func (w *StdoutWriter) reset(lineBreaks int) {
	if lineBreaks == 0 {
		fmt.Fprintf(os.Stdout, "%c[2K\r", ESCAPE)
		return
	}

	for i := 0; i < lineBreaks; i++ {
		fmt.Fprintf(os.Stdout, "%c[%dA", ESCAPE, 1) // move the cursor up
		fmt.Fprintf(os.Stdout, "%c[2K\r", ESCAPE)   // clear the line
	}
}
