package version

import (
	"runtime/debug"
	"time"
)

var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

func init() {
	if info, ok := debug.ReadBuildInfo(); ok {
		if info.Main.Version != "(devel)" && info.Main.Version != "" {
			version = info.Main.Version
		}

		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				if len(setting.Value) > 7 {
					commit = setting.Value[:7]
				} else if setting.Value != "" {
					commit = setting.Value
				}
			case "vcs.time":
				if t, err := time.Parse(time.RFC3339, setting.Value); err == nil {
					date = t.Format("02/01/2006")
				}
			}
		}
	}
}

func Version() string {
	return version
}

func Commit() string {
	return commit
}

func Date() string {
	return date
}
