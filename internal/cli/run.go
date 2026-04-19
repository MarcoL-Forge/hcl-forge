package cli

import (
	"fmt"
	"strings"
)

func Run(args []string) error {
	if len(args) == 0 {
		return usageError("missing command")
	}

	switch args[0] {
	case "render":
		return runRender(args[1:])
	case "help", "-h", "--help":
		return usageError("")
	default:
		return usageError(fmt.Sprintf("unknown command %q", args[0]))
	}
}

func usageError(message string) error {
	usage := strings.TrimSpace(`
usage:
  hcl-forge render --in <file> [--out <file>] [--stdout]

commands:
  render    load a Terraform/HCL file and render it to stdout and/or an output file
`)

	if message == "" {
		return fmt.Errorf("%s", usage)
	}

	return fmt.Errorf("%s\n\n%s", message, usage)
}
