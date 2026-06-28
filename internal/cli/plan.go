package cli

import (
	"flag"

	"github.com/Marc0l95/hclforge/internal/config"
	"github.com/Marc0l95/hclforge/internal/editor"
)

type FilePlan struct {
	SourcePath string
	OutputPath string
	Edits      []editor.Edit
}

type FilePlanResult struct {
	SourcePath string
	OutputPath string
	Results    []editor.EditResult
	Changed    bool
}

type FilePlanJob struct {
	Index int
	Plan  FilePlan
}

func runPlan(args []string) error {
	fs := flag.NewFlagSet("plan", flag.ContinueOnError)

	configPath := fs.String("config", "tfedit.yaml", "path to YAML playbook")

	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}

	plans, err := config.BuildFilePlans(*cfg)
	if err != nil {
		return err
	}

	results, err := editor.PlanFilePlans(plans, cfg.Options.Workers)
	if err != nil {
		return err
	}

	printResults("Plan", results)

	return nil
}
