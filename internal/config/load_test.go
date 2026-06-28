package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_Success(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "tfedit.yaml")

	content := `version: 1
input:
  root_dir: .
  files:
    - main.tf
output:
  mode: overwrite
options:
  workers: 1
  fail_on_no_change: false
edits:
  - type: search_replace
    old: old
    new: new
`

	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Version != 1 {
		t.Fatalf("expected version 1, got %d", cfg.Version)
	}
	if len(cfg.Input.Files) != 1 || cfg.Input.Files[0] != "main.tf" {
		t.Fatalf("unexpected input files: %+v", cfg.Input.Files)
	}
	if len(cfg.Edits) != 1 || cfg.Edits[0].Type != "search_replace" {
		t.Fatalf("unexpected edits: %+v", cfg.Edits)
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "tfedit.yaml")

	if err := os.WriteFile(cfgPath, []byte("version: ["), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatalf("expected parse error, got nil")
	}
}

func TestLoad_InvalidConfig(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "tfedit.yaml")

	content := `version: 1
input:
  root_dir: .
  files: []
output:
  mode: overwrite
edits:
  - type: search_replace
    old: old
    new: new
`

	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatalf("expected validation error, got nil")
	}
}

func TestLoad_ExpandsEnvVars(t *testing.T) {
	t.Setenv("HCL_INPUT_ROOT", "./testing/gke")
	t.Setenv("HCL_OUTPUT_DIR", "./out/from-env")

	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "tfedit.yaml")

	content := `version: 1
input:
  root_dir: ${HCL_INPUT_ROOT}
  files:
    - storage_bucket.tf
output:
  mode: target_dir
  target_dir: ${HCL_OUTPUT_DIR}
options:
  workers: 1
  fail_on_no_change: false
edits:
  - type: search_replace
    old: old
    new: new
`

	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Input.RootDir != "./testing/gke" {
		t.Fatalf("expected expanded input root, got %q", cfg.Input.RootDir)
	}
	if cfg.Output.TargetDir != "./out/from-env" {
		t.Fatalf("expected expanded output dir, got %q", cfg.Output.TargetDir)
	}
}

func TestLoad_ExpandsEnvVarsWithDefault(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "tfedit.yaml")

	content := `version: 1
input:
  root_dir: ${UNSET_ROOT:-./testing/gke}
  files:
    - storage_bucket.tf
output:
  mode: target_dir
  target_dir: ${UNSET_OUTPUT:-./out/default-dir}
options:
  workers: 1
  fail_on_no_change: false
edits:
  - type: search_replace
    old: old
    new: new
`

	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if cfg.Input.RootDir != "./testing/gke" {
		t.Fatalf("expected default root dir, got %q", cfg.Input.RootDir)
	}
	if cfg.Output.TargetDir != "./out/default-dir" {
		t.Fatalf("expected default output dir, got %q", cfg.Output.TargetDir)
	}
}

func TestLoad_MissingRequiredEnvVar(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "tfedit.yaml")

	content := `version: 1
input:
  root_dir: ${MISSING_ROOT}
  files:
    - storage_bucket.tf
output:
  mode: overwrite
options:
  workers: 1
  fail_on_no_change: false
edits:
  - type: search_replace
    old: old
    new: new
`

	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := Load(cfgPath)
	if err == nil {
		t.Fatalf("expected missing env var error, got nil")
	}
}

func TestLoad_PathSelectorConfig(t *testing.T) {
	tmp := t.TempDir()
	cfgPath := filepath.Join(tmp, "tfedit.yaml")

	content := `version: 1
input:
  root_dir: .
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
      description = "managed by hcl-forge"
`

	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if len(cfg.Edits) != 1 || cfg.Edits[0].Block == nil {
		t.Fatalf("expected one edit with block selector")
	}

	if cfg.Edits[0].Block.Path != "resource.google_service_account.nodes" {
		t.Fatalf("unexpected path selector: %q", cfg.Edits[0].Block.Path)
	}
}
