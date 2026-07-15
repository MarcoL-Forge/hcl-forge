package cli

import (
	"fmt"
	"strings"
)

func Run(args []string) error {
	if len(args) < 2 {
		printHelp("")
		return fmt.Errorf("expected command: plan, apply, or version")
	}

	cmd := strings.ToLower(args[1])
	if cmd == "-v" || cmd == "--version" || cmd == "version" {
		printVersion()
		return nil
	}

	if cmd == "-h" || cmd == "--help" || cmd == "help" {
		subcommand := ""
		if len(args) > 2 {
			subcommand = strings.ToLower(args[2])
		}
		printHelp(subcommand)
		return nil
	}

	switch cmd {
	case "plan":
		return runPlan(args[2:])
	case "apply":
		return runApply(args[2:])
	default:
		printHelp("")
		return fmt.Errorf("unknown command: %s", args[1])
	}

}

func printHelp(subcommand string) {
	switch subcommand {
	case "plan":
		fmt.Println(`hcl-forge plan [flags]

Generate a dry-run plan of edits without writing files.

Flags:
  -config string       path to YAML playbook (default "tfedit.yaml")
  -verbose             enable debug logging
  -log-level string    log level: debug|info|warn|error (default "info")
  -log-format string   log format: text|json (default "text")
  -log-output string   log output: stderr|stdout|<file path> (default "stderr")
  -log-artifact string optional NDJSON artifact log file path
  -log-redact string   comma-separated extra keys to redact in logs
  -quiet               suppress human-readable output and emit logs only`)
	case "apply":
		fmt.Println(`hcl-forge apply [flags]

Apply edits and write updated files.

Flags:
  -config string       path to YAML playbook (default "tfedit.yaml")
  -verbose             enable debug logging
  -log-level string    log level: debug|info|warn|error (default "info")
  -log-format string   log format: text|json (default "text")
  -log-output string   log output: stderr|stdout|<file path> (default "stderr")
  -log-artifact string optional NDJSON artifact log file path
  -log-redact string   comma-separated extra keys to redact in logs
  -quiet               suppress human-readable output and emit logs only`)
	default:
		fmt.Println(`hcl-forge - Terraform/HCL bulk editor

Usage:
  hcl-forge <command> [flags]

Commands:
  plan        Generate a dry-run plan without writing files
  apply       Apply edits and write files
	version     Print CLI version
  help        Show help for a command

Examples:
  hcl-forge plan -config examples/easy/playbook.yaml
  hcl-forge apply -config examples/easy/playbook.yaml
	hcl-forge version
  hcl-forge help plan`)
	}
}
