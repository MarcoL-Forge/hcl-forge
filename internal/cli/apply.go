package cli

import (
	"flag"

	"github.com/Marc0l95/hclforge/internal/config"
	"github.com/Marc0l95/hclforge/internal/editor"
)

func runApply(args []string) error {
	fs := flag.NewFlagSet("apply", flag.ContinueOnError)

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

	results, err := editor.ApplyFilePlans(plans, cfg.Options.Workers)
	if err != nil {
		return err
	}

	printResults("Apply", results)

	return nil
}