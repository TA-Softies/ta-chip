package version

import "runtime/debug"

// Version is injected at build time via:
//
//	go build -ldflags="-X ta-chip/version.Version=v1.2.3"
//
// When running locally without ldflags it falls back to the git commit hash.
var Version = "dev"

func init() {
	if Version != "dev" {
		return
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	rev := ""
	dirty := false
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			if len(s.Value) >= 7 {
				rev = s.Value[:7]
			}
		case "vcs.modified":
			dirty = s.Value == "true"
		}
	}
	if rev != "" {
		Version = "dev-" + rev
		if dirty {
			Version += "-dirty"
		}
	}
}
