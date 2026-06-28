package cli

import (
	"fmt"
)

func Run(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf("expected command: plan or apply")
	}

	switch args[1] {
	case "plan":
		return runPlan(args[2:])
	case "apply":
		return runApply(args[2:])
	default:
		return fmt.Errorf("unknown command: %s", args[1])
	}

}
