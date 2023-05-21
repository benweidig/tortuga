package version

import (
	"fmt"
	"time"
)

var (
	// Version of the app
	Version = "2.3.3"

	// CommitHash is the commit this version was built on, needs to be set by the linker
	CommitHash = "n/a"

	// CompileDate is the date this binary was compiled on
	CompileDate = ""
)

// BuildVersion combines available information to a nicer looking version string
func BuildVersion() string {
	var date = CompileDate
	if len(date) == 0 {
		date = time.Now().String()
	}
	return fmt.Sprintf("%s-%s (%s)", Version, CommitHash, date)
}
