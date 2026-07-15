package main

import (
	"fmt"
	"os"

	"github.com/MarcoL-Forge/hcl-forge/internal/cli"
	"github.com/MarcoL-Forge/hcl-forge/internal/logging"
)

func run(args []string) error {
	return cli.Run(args)
}

func main() {
	if err := run(os.Args); err != nil {
		logging.Default().Error("cli_exit_error", map[string]any{"error": err.Error()})
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
