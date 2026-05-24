// Package version exposes build-time metadata. Version, CommitHash, and
// CompileDate are injected by the linker via -ldflags during release builds;
// they fall back to sensible dev defaults when building locally.
package version

import (
	"fmt"
	"time"
)

var (
	// Version of the app
	Version = "3.0.2"

	// CommitHash is the commit this version was built on, needs to be set by the linker
	CommitHash = "dev"

	// CompileDate is the date this binary was compiled on
	CompileDate = ""
)

// BuildVersion combines available information to a nicer looking version string
func BuildVersion() string {
	var date = CompileDate
	if len(date) == 0 {
		date = time.Now().Format("2006-01-02T15:04:05")
	}
	return fmt.Sprintf("%s-%s (%s)", Version, CommitHash, date)
}
