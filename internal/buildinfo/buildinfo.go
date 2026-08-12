package buildinfo

import "fmt"

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

// String returns the build identity embedded by the release pipeline.
func String() string {
	return fmt.Sprintf("cyberpilot %s (commit %s, built %s)", version, commit, date)
}
