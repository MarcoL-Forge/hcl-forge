package cli

import "fmt"

// Version is the CLI version and can be overridden at build time via -ldflags.
var Version = "dev"

func printVersion() {
	fmt.Printf("hcl-forge version %s\n", Version)
}
