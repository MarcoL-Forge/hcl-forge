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
	case "replace":
		return runReplace(args[1:])
	case "apply":
		return runApply(args[1:])
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
	hcl-forge replace --in <file> [--target <selector> | --block-type <type> [--labels <a,b>] --attr <name>] --value <value> [--value-type <type>] [--out <file>] [--stdout]
	hcl-forge apply --playbook <file> [--out <file>] [--stdout]

commands:
  render    load a Terraform/HCL file and render it to stdout and/or an output file
	replace   replace a targeted attribute value and write the updated Terraform output
	apply     apply a YAML playbook to a Terraform/HCL file
`)

	if message == "" {
		return fmt.Errorf("%s", usage)
	}

	return fmt.Errorf("%s\n\n%s", message, usage)
}
