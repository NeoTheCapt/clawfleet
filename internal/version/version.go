package version

import (
	"fmt"
	"strings"
)

// These are set via -ldflags at build time.
var (
	Component = "unknown" // server | node | fleetctl
	Version   = "dev"     // semantic version/tag
	Commit    = ""        // git commit sha
	BuildTime = ""        // RFC3339
)

type Info struct {
	Component string `json:"component"`
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	BuildTime string `json:"build_time"`
}

func Get() Info {
	return Info{Component: Component, Version: Version, Commit: Commit, BuildTime: BuildTime}
}

func String() string {
	i := Get()
	c := i.Commit
	if len(c) > 8 {
		c = c[:8]
	}
	parts := []string{i.Component, i.Version}
	if c != "" {
		parts = append(parts, fmt.Sprintf("(%s)", c))
	}
	if i.BuildTime != "" {
		parts = append(parts, i.BuildTime)
	}
	return strings.Join(parts, " ")
}
