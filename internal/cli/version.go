package cli

import (
	"fmt"
	"runtime/debug"
)

// Version is the CLI version and can be overridden at build time via -ldflags.
var Version = "dev"

var readBuildInfo = debug.ReadBuildInfo

func resolvedVersion() string {
	if Version != "" && Version != "dev" {
		return Version
	}

	if info, ok := readBuildInfo(); ok {
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			return info.Main.Version
		}
	}

	if Version == "" {
		return "dev"
	}

	return Version
}

func printVersion() {
	fmt.Printf("hcl-forge version %s\n", resolvedVersion())
}
