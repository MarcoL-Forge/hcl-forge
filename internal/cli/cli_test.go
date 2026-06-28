package cli

import (
	"os"
	"path/filepath"
	"strings"
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

func TestRun_HelpCommand(t *testing.T) {
	err := Run([]string{"hcl-forge", "help"})
	if err != nil {
		t.Fatalf("expected help command to succeed, got: %v", err)
	}
}

func TestRun_HelpSubcommands(t *testing.T) {
	for _, args := range [][]string{
		{"hcl-forge", "help", "plan"},
		{"hcl-forge", "help", "apply"},
		{"hcl-forge", "--help"},
		{"hcl-forge", "-h"},
	} {
		if err := Run(args); err != nil {
			t.Fatalf("expected %v to succeed, got: %v", args, err)
		}
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

func TestRun_PlanFailOnNoChange(t *testing.T) {
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
		"  fail_on_no_change: true\n" +
		"edits:\n" +
		"  - type: search_replace\n" +
		"    old: does-not-exist\n" +
		"    new: new\n"
	if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	err := Run([]string{"hcl-forge", "plan", "-config", configPath})
	if err == nil {
		t.Fatalf("expected plan to fail when fail_on_no_change is true")
	}

	if !strings.Contains(err.Error(), "fail_on_no_change") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRun_ApplyFailOnNoChange(t *testing.T) {
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
		"  fail_on_no_change: true\n" +
		"edits:\n" +
		"  - type: search_replace\n" +
		"    old: does-not-exist\n" +
		"    new: new\n"
	if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	err := Run([]string{"hcl-forge", "apply", "-config", configPath})
	if err == nil {
		t.Fatalf("expected apply to fail when fail_on_no_change is true")
	}

	if !strings.Contains(err.Error(), "fail_on_no_change") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRun_ApplyWithPathSelector(t *testing.T) {
	tmp := t.TempDir()
	inputPath := filepath.Join(tmp, "main.tf")
	input := `resource "google_service_account" "nodes" {
  account_id = "nodes"
}
`
	if err := os.WriteFile(inputPath, []byte(input), 0o644); err != nil {
		t.Fatalf("write input file: %v", err)
	}

	configPath := filepath.Join(tmp, "tfedit.yaml")
	config := `version: 1
input:
  root_dir: ` + tmp + `
  files:
    - main.tf
output:
  mode: overwrite
options:
  workers: 1
  fail_on_no_change: false
edits:
  - type: insert_hcl
    block:
      path: resource.google_service_account.nodes
    hcl: |
      description = "managed"
`
	if err := os.WriteFile(configPath, []byte(config), 0o644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	if err := Run([]string{"hcl-forge", "apply", "-config", configPath}); err != nil {
		t.Fatalf("apply command failed: %v", err)
	}

	updated, err := os.ReadFile(inputPath)
	if err != nil {
		t.Fatalf("read updated file: %v", err)
	}

	out := string(updated)
	if !strings.Contains(out, "description") || !strings.Contains(out, "managed") {
		t.Fatalf("unexpected updated content: %q", out)
	}
}
