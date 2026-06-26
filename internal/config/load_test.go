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
