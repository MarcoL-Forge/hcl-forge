package config

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/Marc0l95/hclforge/internal/editor"
)

func TestBuildFilePlans_Overwrite(t *testing.T) {
	root := t.TempDir()
	cfg := Config{
		Version: 1,
		Input: InputConfig{
			RootDir: root,
			Files:   []string{"a.tf", "nested/b.tf"},
		},
		Output: OutputConfig{Mode: "overwrite"},
		Edits: []EditConfig{{
			Type: "search_replace",
			Old:  "old",
			New:  "new",
		}},
	}

	plans, err := BuildFilePlans(cfg)
	if err != nil {
		t.Fatalf("build file plans: %v", err)
	}

	if len(plans) != 2 {
		t.Fatalf("expected 2 plans, got %d", len(plans))
	}

	for _, p := range plans {
		if p.SourcePath != p.OutputPath {
			t.Fatalf("overwrite mode expected source=output, got source=%q output=%q", p.SourcePath, p.OutputPath)
		}
		if len(p.Edits) != 1 {
			t.Fatalf("expected one edit, got %d", len(p.Edits))
		}
	}
}

func TestBuildFilePlans_TargetDir(t *testing.T) {
	root := t.TempDir()
	target := t.TempDir()

	cfg := Config{
		Version: 1,
		Input: InputConfig{
			RootDir: root,
			Files:   []string{"a.tf", "nested/b.tf"},
		},
		Output: OutputConfig{Mode: "target_dir", TargetDir: target},
		Edits: []EditConfig{{
			Type: "search_replace",
			Old:  "old",
			New:  "new",
		}},
	}

	plans, err := BuildFilePlans(cfg)
	if err != nil {
		t.Fatalf("build file plans: %v", err)
	}

	if len(plans) != 2 {
		t.Fatalf("expected 2 plans, got %d", len(plans))
	}

	want := []struct {
		source string
		output string
	}{
		{source: filepath.Join(root, "a.tf"), output: filepath.Join(target, "a.tf")},
		{source: filepath.Join(root, "nested", "b.tf"), output: filepath.Join(target, "nested", "b.tf")},
	}

	for i := range want {
		if plans[i].SourcePath != want[i].source {
			t.Fatalf("plan %d source mismatch: got %q want %q", i, plans[i].SourcePath, want[i].source)
		}
		if plans[i].OutputPath != want[i].output {
			t.Fatalf("plan %d output mismatch: got %q want %q", i, plans[i].OutputPath, want[i].output)
		}
	}
}

func TestBuildFilePlans_UnsupportedEditType(t *testing.T) {
	cfg := Config{
		Version: 1,
		Input: InputConfig{
			RootDir: ".",
			Files:   []string{"main.tf"},
		},
		Output: OutputConfig{Mode: "overwrite"},
		Edits:  []EditConfig{{Type: "unknown"}},
	}

	_, err := BuildFilePlans(cfg)
	if err == nil {
		t.Fatalf("expected error for unsupported edit type")
	}
	if !strings.Contains(err.Error(), "unsupported edit type") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildFilePlans_UnsupportedOutputMode(t *testing.T) {
	cfg := Config{
		Version: 1,
		Input: InputConfig{
			RootDir: ".",
			Files:   []string{"main.tf"},
		},
		Output: OutputConfig{Mode: "invalid"},
		Edits: []EditConfig{{
			Type: "search_replace",
			Old:  "old",
			New:  "new",
		}},
	}

	_, err := BuildFilePlans(cfg)
	if err == nil {
		t.Fatalf("expected error for unsupported output mode")
	}
	if !strings.Contains(err.Error(), "unsupported output mode") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuildFilePlans_InsertHCLEdit(t *testing.T) {
	root := t.TempDir()
	cfg := Config{
		Version: 1,
		Input: InputConfig{
			RootDir: root,
			Files:   []string{"main.tf"},
		},
		Output: OutputConfig{Mode: "overwrite"},
		Edits: []EditConfig{{
			Type: "insert_hcl",
			HCL:  "force_destroy = true",
			Block: &BlockSelector{
				BlockType: "resource",
				Labels:    []string{"google_storage_bucket", "bucket"},
			},
		}},
	}

	plans, err := BuildFilePlans(cfg)
	if err != nil {
		t.Fatalf("build file plans: %v", err)
	}

	if len(plans) != 1 {
		t.Fatalf("expected 1 plan, got %d", len(plans))
	}

	if len(plans[0].Edits) != 1 {
		t.Fatalf("expected 1 edit, got %d", len(plans[0].Edits))
	}

	insertEdit, ok := plans[0].Edits[0].(editor.InsertHCLEdit)
	if !ok {
		t.Fatalf("expected InsertHCLEdit, got %T", plans[0].Edits[0])
	}

	if insertEdit.HCL != "force_destroy = true" {
		t.Fatalf("unexpected insert hcl content: %q", insertEdit.HCL)
	}

	if insertEdit.TargetBlock == nil || insertEdit.TargetBlock.Type != "resource" {
		t.Fatalf("unexpected target block: %+v", insertEdit.TargetBlock)
	}
}
