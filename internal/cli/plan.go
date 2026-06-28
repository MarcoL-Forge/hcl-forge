package cli

import (
	"errors"
	"flag"
	"fmt"

	"github.com/Marc0l95/hclforge/internal/config"
	"github.com/Marc0l95/hclforge/internal/editor"
	"github.com/Marc0l95/hclforge/internal/logging"
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
	verbose := fs.Bool("verbose", false, "enable debug logging")
	logLevel := fs.String("log-level", "info", "log level: debug|info|warn|error")
	logFormat := fs.String("log-format", "text", "log format: text|json")
	logOutput := fs.String("log-output", "stderr", "log output: stderr|stdout|<file path>")
	logArtifact := fs.String("log-artifact", "", "optional NDJSON artifact log file path")
	logRedact := fs.String("log-redact", "", "comma-separated extra keys to redact in logs")
	quiet := fs.Bool("quiet", false, "suppress human-readable output and emit logs only")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	logger, closer, err := logging.New(logging.Config{
		Verbose:    *verbose,
		Level:      *logLevel,
		Format:     *logFormat,
		Output:     *logOutput,
		Artifact:   *logArtifact,
		RedactKeys: *logRedact,
	})
	if err != nil {
		return err
	}
	if closer != nil {
		defer closer.Close()
	}
	logging.SetDefault(logger)

	logger.Info("plan_start", map[string]any{"config": *configPath})

	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Error("plan_config_load_failed", map[string]any{"error": err.Error()})
		return err
	}
	logger.Debug("plan_config_loaded", map[string]any{"workers": cfg.Options.Workers})

	plans, err := config.BuildFilePlans(*cfg)
	if err != nil {
		logger.Error("plan_build_failed", map[string]any{"error": err.Error()})
		return err
	}
	logger.Info("plan_built", map[string]any{"files": len(plans)})

	results, err := editor.PlanFilePlans(plans, cfg.Options.Workers)
	printResults("Plan", results, logger, *quiet)
	if err != nil {
		logger.Error("plan_failed", map[string]any{"error": err.Error()})
		return err
	}

	if cfg.Options.FailOnNoChange && !anyFileChanged(results) {
		logger.Error("plan_no_change_failure", map[string]any{"fail_on_no_change": true})
		return fmt.Errorf("fail_on_no_change enabled and no changes were produced")
	}

	logger.Info("plan_completed", map[string]any{"files": len(results)})

	return nil
}

func anyFileChanged(results []editor.FilePlanResult) bool {
	for _, result := range results {
		if result.Changed {
			return true
		}
	}

	return false
}
