package build

import (
	"runtime/debug"
)

// Version is set via -ldflags; falls back to VCS metadata (go install).
var Version = "development" //nolint:gochecknoglobals

// Commit is set via -ldflags; falls back to VCS metadata (go install).
var Commit = "unknown" //nolint:gochecknoglobals

// Date is set via -ldflags; falls back to VCS metadata (go install).
var Date = "" //nolint:gochecknoglobals

func init() { //nolint:gochecknoinits
	if Commit != "unknown" && Date != "" {
		return
	}

	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}

	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			if Commit == "unknown" && len(s.Value) > 6 {
				Commit = s.Value[:7]
			}
		case "vcs.time":
			if Date == "" {
				Date = s.Value
			}
		}
	}
}
