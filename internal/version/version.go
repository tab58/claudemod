package version

import (
	"fmt"
	"runtime"
)

// Injected at build time via -ldflags "-X ...".
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// Info holds the build and runtime version metadata.
type Info struct {
	Version string
	Commit  string
	Date    string
	Go      string
	OS      string
	Arch    string
}

// Get returns a copy of the current version info populated from
// build-time ldflags and runtime constants.
func Get() Info {
	return Info{
		Version: version,
		Commit:  commit,
		Date:    date,
		Go:      runtime.Version(),
		OS:      runtime.GOOS,
		Arch:    runtime.GOARCH,
	}
}

// String formats the version info as a single human-readable line.
func (i Info) String() string {
	return fmt.Sprintf("claudemod %s (commit: %s, built: %s, %s, %s/%s)",
		i.Version, i.Commit, i.Date, i.Go, i.OS, i.Arch)
}
