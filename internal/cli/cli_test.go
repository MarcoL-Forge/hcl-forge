package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRun_RequiresCommand(t *testing.T) {
	err := Run([]string{"hcl-forge"})
	if err == nil {
		t.Fatalf("expected error when command is missing")
	}
}

func TestRun_UnknownCommand(t *testing.T) {
	err := Run([]string{"hcl-forge", "unknown"})
	if err == nil {
		t.Fatalf("expected error for unknown command")
	}
}

func TestRun_PlanAndApplyWithConfig(t *testing.T) {
	tmp := t.TempDir()
	inputPath := filepath.Join(tmp, "main.tf")
	if err := os.WriteFile(inputPath, []byte("x = \"old\"\n"), 0o644); err != nil {
		t.Fatalf("write input file: %v", err)
	}

	configPath := filepath.Join(tmp, "tfedit.yaml")
	config := "version: 1\n" +
		"input:\n" +
		"  root_dir: \"" + tmp + "\"\n" +
		"  files:\n" +
		"    - main.tf\n" +
		"output:\n" +
		"  mode: overwrite\n" +
		"options:\n" +
		"  workers: 1\n" +
		"  fail_on_no_change: false\n" +
		"edits:\n" +
		"  - type: search_replace\n" +
		"    old: old\n" +
		"    new: new\n"
	if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	if err := Run([]string{"hcl-forge", "plan", "-config", configPath}); err != nil {
		t.Fatalf("plan command failed: %v", err)
	}

	if err := Run([]string{"hcl-forge", "apply", "-config", configPath}); err != nil {
		t.Fatalf("apply command failed: %v", err)
	}

	updated, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatalf("read updated file: %v", err)
	}
	if string(updated) != "x = \"new\"\n" {
		t.Fatalf("unexpected updated content: %q", string(updated))
	}
}
