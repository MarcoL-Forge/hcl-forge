package config

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/MarcoL-Forge/hcl-forge/internal/editor"
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

	if len(insertEdit.TargetBlock.Parents) != 0 {
		t.Fatalf("expected no parents for target block, got %+v", insertEdit.TargetBlock.Parents)
	}
}

func TestBuildFilePlans_DeleteHCLEdit(t *testing.T) {
	root := t.TempDir()
	cfg := Config{
		Version: 1,
		Input: InputConfig{
			RootDir: root,
			Files:   []string{"main.tf"},
		},
		Output: OutputConfig{Mode: "overwrite"},
		Edits: []EditConfig{{
			Type:      "delete_hcl",
			Attribute: "location",
			DeleteAll: true,
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

	deleteEdit, ok := plans[0].Edits[0].(editor.DeleteHCLEdit)
	if !ok {
		t.Fatalf("expected DeleteHCLEdit, got %T", plans[0].Edits[0])
	}

	if deleteEdit.Attribute != "location" {
		t.Fatalf("unexpected attribute: %q", deleteEdit.Attribute)
	}

	if !deleteEdit.DeleteAll {
		t.Fatalf("expected delete_all to be true")
	}

	if deleteEdit.KeepOnly {
		t.Fatalf("expected keep_only to be false")
	}

	if deleteEdit.MatchMode != "" {
		t.Fatalf("expected empty match_mode by default, got %q", deleteEdit.MatchMode)
	}

	if deleteEdit.TargetBlock == nil || deleteEdit.TargetBlock.Type != "resource" {
		t.Fatalf("unexpected target block: %+v", deleteEdit.TargetBlock)
	}
}

func TestBuildFilePlans_DeleteHCLEditKeepOnlyAndRegexMode(t *testing.T) {
	root := t.TempDir()
	cfg := Config{
		Version: 1,
		Input: InputConfig{
			RootDir: root,
			Files:   []string{"main.tf"},
		},
		Output: OutputConfig{Mode: "overwrite"},
		Edits: []EditConfig{{
			Type:      "delete_hcl",
			KeepOnly:  true,
			MatchMode: "regex",
			Block: &BlockSelector{
				BlockType: "resource",
				Labels:    []string{"tfe_workspace", "example(1|3)"},
			},
		}},
	}

	plans, err := BuildFilePlans(cfg)
	if err != nil {
		t.Fatalf("build file plans: %v", err)
	}

	deleteEdit, ok := plans[0].Edits[0].(editor.DeleteHCLEdit)
	if !ok {
		t.Fatalf("expected DeleteHCLEdit, got %T", plans[0].Edits[0])
	}

	if !deleteEdit.KeepOnly {
		t.Fatalf("expected keep_only to be true")
	}

	if deleteEdit.MatchMode != "regex" {
		t.Fatalf("expected match_mode regex, got %q", deleteEdit.MatchMode)
	}
}

func TestBuildFilePlans_NestedParentsMapping(t *testing.T) {
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
			HCL:  "disk_size_gb = 100",
			Block: &BlockSelector{
				BlockType: "shielded_instance_config",
				Labels:    []string{},
				Parents: []ParentSelector{
					{BlockType: "resource", Labels: []string{"google_container_node_pool", "pool"}},
					{BlockType: "node_config", Labels: []string{}},
				},
			},
		}},
	}

	plans, err := BuildFilePlans(cfg)
	if err != nil {
		t.Fatalf("build file plans: %v", err)
	}

	insertEdit, ok := plans[0].Edits[0].(editor.InsertHCLEdit)
	if !ok {
		t.Fatalf("expected InsertHCLEdit, got %T", plans[0].Edits[0])
	}

	if insertEdit.TargetBlock == nil {
		t.Fatalf("expected target block")
	}

	if len(insertEdit.TargetBlock.Parents) != 2 {
		t.Fatalf("expected 2 parent selectors, got %d", len(insertEdit.TargetBlock.Parents))
	}

	if insertEdit.TargetBlock.Parents[0].Type != "resource" {
		t.Fatalf("unexpected first parent type: %q", insertEdit.TargetBlock.Parents[0].Type)
	}

	if insertEdit.TargetBlock.Parents[1].Type != "node_config" {
		t.Fatalf("unexpected second parent type: %q", insertEdit.TargetBlock.Parents[1].Type)
	}
}

func TestBuildFilePlans_PathSelectorMapping(t *testing.T) {
	root := t.TempDir()
	cfg := Config{
		Version: 1,
		Input: InputConfig{
			RootDir: root,
			Files:   []string{"main.tf"},
		},
		Output: OutputConfig{Mode: "overwrite"},
		Edits: []EditConfig{{
			Type: "delete_hcl",
			Block: &BlockSelector{
				Path: "resource.google_service_account.nodes",
			},
		}},
	}

	plans, err := BuildFilePlans(cfg)
	if err != nil {
		t.Fatalf("build file plans: %v", err)
	}

	deleteEdit, ok := plans[0].Edits[0].(editor.DeleteHCLEdit)
	if !ok {
		t.Fatalf("expected DeleteHCLEdit, got %T", plans[0].Edits[0])
	}

	if deleteEdit.TargetBlock == nil {
		t.Fatalf("expected target block")
	}

	if deleteEdit.TargetBlock.Type != "resource" {
		t.Fatalf("unexpected target type: %q", deleteEdit.TargetBlock.Type)
	}

	if len(deleteEdit.TargetBlock.Labels) != 2 || deleteEdit.TargetBlock.Labels[0] != "google_service_account" || deleteEdit.TargetBlock.Labels[1] != "nodes" {
		t.Fatalf("unexpected target labels: %+v", deleteEdit.TargetBlock.Labels)
	}
}

func TestBuildFilePlans_InsertHCLEditEnsureAndGuard(t *testing.T) {
	root := t.TempDir()
	cfg := Config{
		Version: 1,
		Input:   InputConfig{RootDir: root, Files: []string{"main.tf"}},
		Output:  OutputConfig{Mode: "overwrite"},
		Edits: []EditConfig{{
			Type:              "insert_hcl",
			HCL:               "description = \"managed\"",
			EnsureTargetBlock: true,
			Guard:             &GuardConfig{IfTargetMissing: true},
			Block: &BlockSelector{
				Path: "resource.google_service_account.nodes",
			},
		}},
	}

	plans, err := BuildFilePlans(cfg)
	if err != nil {
		t.Fatalf("build file plans: %v", err)
	}

	insertEdit, ok := plans[0].Edits[0].(editor.InsertHCLEdit)
	if !ok {
		t.Fatalf("expected InsertHCLEdit, got %T", plans[0].Edits[0])
	}

	if !insertEdit.EnsureTargetBlock {
		t.Fatalf("expected ensure_target_block=true")
	}

	if insertEdit.Guard == nil || !insertEdit.Guard.IfTargetMissing {
		t.Fatalf("expected guard.if_target_missing=true")
	}
}

func TestBuildFilePlans_SetAttributeHCLEdit(t *testing.T) {
	root := t.TempDir()
	cfg := Config{
		Version: 1,
		Input:   InputConfig{RootDir: root, Files: []string{"main.tf"}},
		Output:  OutputConfig{Mode: "overwrite"},
		Edits: []EditConfig{{
			Type:            "set_attribute",
			Attribute:       "force_destroy",
			ValueHCL:        "true",
			CreateIfMissing: true,
			Block: &BlockSelector{
				Path: "resource.google_storage_bucket.bucket",
			},
		}},
	}

	plans, err := BuildFilePlans(cfg)
	if err != nil {
		t.Fatalf("build file plans: %v", err)
	}

	setEdit, ok := plans[0].Edits[0].(editor.SetAttributeHCLEdit)
	if !ok {
		t.Fatalf("expected SetAttributeHCLEdit, got %T", plans[0].Edits[0])
	}

	if setEdit.Attribute != "force_destroy" {
		t.Fatalf("unexpected attribute: %q", setEdit.Attribute)
	}
	if setEdit.ValueHCL != "true" {
		t.Fatalf("unexpected value_hcl: %q", setEdit.ValueHCL)
	}
	if !setEdit.CreateIfMissing {
		t.Fatalf("expected create_if_missing to be true")
	}
	if setEdit.TargetBlock == nil || setEdit.TargetBlock.Type != "resource" {
		t.Fatalf("unexpected target block: %+v", setEdit.TargetBlock)
	}
}
