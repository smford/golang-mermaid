package mermaid

import (
	"runtime/debug"
	"strings"
)

// Version represents the current Semantic Version (SemVer) of the golang-mermaid module.
var Version = "1.1.1"

func init() {
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, dep := range info.Deps {
			if dep.Path == "github.com/smford/golang-mermaid" && dep.Version != "" && dep.Version != "(devel)" {
				Version = strings.TrimPrefix(dep.Version, "v")
				return
			}
		}
		if info.Main.Path == "github.com/smford/golang-mermaid" && info.Main.Version != "" && info.Main.Version != "(devel)" {
			Version = strings.TrimPrefix(info.Main.Version, "v")
		}
	}
}
