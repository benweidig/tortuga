package version

import (
	"fmt"
	"time"
)

var (
	// Version of the app, should be set by the linker when compiling
	Version = "1.2.0"

	// CommitHash is the commit this version was built on
	CommitHash = "n/a"

	// CompileDate is the date this binary was compiled on
	CompileDate = ""
)

func BuildVersion() string {
	var date = CompileDate
	if len(date) == 0 {
		date = time.Now().String()
	}
	return fmt.Sprintf("%s-%s (%s)", Version, CommitHash, CompileDate)
}
