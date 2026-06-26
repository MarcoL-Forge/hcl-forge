//go:build e2e

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIApply_E2E(t *testing.T) {
	tmp := t.TempDir()
	inputDir := filepath.Join(tmp, "input")
	outputDir := filepath.Join(tmp, "output")

	if err := os.MkdirAll(inputDir, 0o755); err != nil {
		t.Fatalf("create input dir: %v", err)
	}

	inputFile := filepath.Join(inputDir, "main.tf")
	input := []byte("resource \"null_resource\" \"x\" {\n  triggers = { v = \"old\" }\n}\n")
	if err := os.WriteFile(inputFile, input, 0o644); err != nil {
		t.Fatalf("write input file: %v", err)
	}

	cfgPath := filepath.Join(tmp, "tfedit.yaml")
	cfg := fmt.Sprintf(`version: 1
input:
  root_dir: %q
  files:
    - main.tf
output:
  mode: target_dir
  target_dir: %q
options:
  workers: 1
  fail_on_no_change: false
edits:
  - type: search_replace
    old: old
    new: new
`, inputDir, outputDir)

	if err := os.WriteFile(cfgPath, []byte(cfg), 0o644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get cwd: %v", err)
	}

	repoRoot := filepath.Clean(filepath.Join(cwd, "..", ".."))
	cmd := exec.Command("go", "run", "./cmd/hcl-forge", "apply", "-config", cfgPath)
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("run cli apply: %v\noutput:\n%s", err, string(out))
	}

	outFile := filepath.Join(outputDir, "main.tf")
	data, err := os.ReadFile(outFile)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}

	if string(data) == string(input) {
		t.Fatalf("expected output to be modified")
	}

	if !strings.Contains(string(data), "new") {
		t.Fatalf("expected output to contain replacement, got:\n%s", string(data))
	}
}
