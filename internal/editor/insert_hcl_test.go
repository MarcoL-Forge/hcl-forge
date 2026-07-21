package editor

import (
	"strings"
	"testing"
)

func TestInsertHCLEdit_Apply_ToRootBlock(t *testing.T) {
	input := `resource "google_storage_bucket" "bucket" {
  name     = "my-bucket"
  location = "europe-west1"
}
`

	edit := InsertHCLEdit{
		HCL: `variable "project_id" {
  type = string
}
`,
	}

	updated, result, err := edit.Apply([]byte(input))
	if err != nil {
		t.Fatalf("apply insert_hcl: %v", err)
	}
	if !result.Changed {
		t.Fatalf("expected change")
	}

	out := string(updated)
	if !strings.Contains(out, `variable "project_id"`) {
		t.Fatalf("expected inserted variable block, got:\n%s", out)
	}
}

func TestInsertHCLEdit_Apply_ToTargetBlock(t *testing.T) {
	input := `resource "google_storage_bucket" "bucket" {
  name     = "my-bucket"
  location = "europe-west1"
}
`

	edit := InsertHCLEdit{
		HCL: `force_destroy = true`,
		TargetBlock: &BlockSelector{
			Type:   "resource",
			Labels: []string{"google_storage_bucket", "bucket"},
		},
	}

	updated, result, err := edit.Apply([]byte(input))
	if err != nil {
		t.Fatalf("apply insert_hcl in block: %v", err)
	}
	if !result.Changed {
		t.Fatalf("expected change")
	}

	out := string(updated)
	if !strings.Contains(out, `force_destroy = true`) {
		t.Fatalf("expected inserted attribute in target block, got:\n%s", out)
	}
}

func TestInsertHCLEdit_TargetBlockNotFound(t *testing.T) {
	input := `resource "google_storage_bucket" "bucket" {}`

	edit := InsertHCLEdit{
		HCL: `force_destroy = true`,
		TargetBlock: &BlockSelector{
			Type:   "resource",
			Labels: []string{"google_storage_bucket", "missing"},
		},
	}

	_, _, err := edit.Apply([]byte(input))
	if err == nil {
		t.Fatalf("expected error when target block is missing")
	}
}

func TestInsertHCLEdit_Apply_ToRootPreservesBlockSeparation(t *testing.T) {
	input := `resource "google_storage_bucket" "bucket" {
  name = "my-bucket"
}

variable "project_id" {
  type = string
}
`

	edit := InsertHCLEdit{
		HCL: `terraform {
  required_version = ">= 1.5.0"
}`,
	}

	updated, _, err := edit.Apply([]byte(input))
	if err != nil {
		t.Fatalf("apply insert_hcl: %v", err)
	}

	out := string(updated)
	if strings.Contains(out, "} terraform {") {
		t.Fatalf("expected root block insertion to keep newline separation, got:\n%s", out)
	}
}

func TestInsertHCLEdit_Apply_ToDeepNestedBlockWithParents(t *testing.T) {
	input := `resource "google_container_node_pool" "pool" {
  node_config {
		shielded_instance_config {
			enable_secure_boot = true
    }
  }

  upgrade_settings {
    max_surge = 1
  }
}
`

	edit := InsertHCLEdit{
		HCL: `disk_size_gb = 100`,
		TargetBlock: &BlockSelector{
			Type:   "shielded_instance_config",
			Labels: []string{},
			Parents: []ParentSelector{
				{Type: "resource", Labels: []string{"google_container_node_pool", "pool"}},
				{Type: "node_config", Labels: []string{}},
			},
		},
	}

	updated, result, err := edit.Apply([]byte(input))
	if err != nil {
		t.Fatalf("apply insert_hcl in deep block: %v", err)
	}
	if !result.Changed {
		t.Fatalf("expected change")
	}

	out := string(updated)
	if !strings.Contains(out, "disk_size_gb") || !strings.Contains(out, "100") {
		t.Fatalf("expected inserted attribute in deeply nested target block, got:\n%s", out)
	}
}

func TestInsertHCLEdit_EnsureTargetBlock_CreatesMissingTarget(t *testing.T) {
	input := `resource "google_container_node_pool" "pool" {
  node_config {}
}
`

	edit := InsertHCLEdit{
		HCL:               `enable_secure_boot = true`,
		EnsureTargetBlock: true,
		TargetBlock: &BlockSelector{
			Type:   "shielded_instance_config",
			Labels: []string{},
			Parents: []ParentSelector{
				{Type: "resource", Labels: []string{"google_container_node_pool", "pool"}},
				{Type: "node_config", Labels: []string{}},
			},
		},
	}

	updated, result, err := edit.Apply([]byte(input))
	if err != nil {
		t.Fatalf("apply insert_hcl ensure target: %v", err)
	}
	if !result.Changed {
		t.Fatalf("expected changed result")
	}

	out := string(updated)
	if !strings.Contains(out, "shielded_instance_config") {
		t.Fatalf("expected created target block, got:\n%s", out)
	}
	if !strings.Contains(out, "enable_secure_boot") {
		t.Fatalf("expected inserted attribute in created block, got:\n%s", out)
	}
}

func TestInsertHCLEdit_GuardIfTargetExists_SkipsWhenMissing(t *testing.T) {
	input := `resource "google_storage_bucket" "bucket" {}`

	edit := InsertHCLEdit{
		HCL: `force_destroy = true`,
		TargetBlock: &BlockSelector{
			Type:   "resource",
			Labels: []string{"google_storage_bucket", "missing"},
		},
		Guard: &InsertGuard{IfTargetExists: true},
	}

	updated, result, err := edit.Apply([]byte(input))
	if err != nil {
		t.Fatalf("expected guard skip, got error: %v", err)
	}
	if result.Changed {
		t.Fatalf("expected no change when guard skips")
	}
	if string(updated) != input {
		t.Fatalf("expected unchanged input when guard skips")
	}
}

func TestInsertHCLEdit_GuardIfTargetMissing_SkipsWhenExists(t *testing.T) {
	input := `resource "google_storage_bucket" "bucket" {
  name = "my-bucket"
}
`

	edit := InsertHCLEdit{
		HCL: `force_destroy = true`,
		TargetBlock: &BlockSelector{
			Type:   "resource",
			Labels: []string{"google_storage_bucket", "bucket"},
		},
		Guard: &InsertGuard{IfTargetMissing: true},
	}

	updated, result, err := edit.Apply([]byte(input))
	if err != nil {
		t.Fatalf("expected guard skip, got error: %v", err)
	}
	if result.Changed {
		t.Fatalf("expected no change when guard skips")
	}
	if string(updated) != input {
		t.Fatalf("expected unchanged input when guard skips")
	}
}

func TestInsertHCLEdit_BlockSnippet_IsIdempotent(t *testing.T) {
	input := `resource "google_storage_bucket" "bucket" {
  name = "my-bucket"
}
`

	edit := InsertHCLEdit{
		HCL: `versioning {
  enabled = true
}`,
		TargetBlock: &BlockSelector{
			Type:   "resource",
			Labels: []string{"google_storage_bucket", "bucket"},
		},
	}

	firstOut, firstResult, err := edit.Apply([]byte(input))
	if err != nil {
		t.Fatalf("first apply insert_hcl: %v", err)
	}
	if !firstResult.Changed {
		t.Fatalf("expected first apply to change file")
	}

	secondOut, secondResult, err := edit.Apply(firstOut)
	if err != nil {
		t.Fatalf("second apply insert_hcl: %v", err)
	}
	if secondResult.Changed {
		t.Fatalf("expected second apply to be idempotent")
	}

	out := string(secondOut)
	if strings.Count(out, "versioning {") != 1 {
		t.Fatalf("expected single versioning block after rerun, got:\n%s", out)
	}
}

func TestApplyEdits_InsertScopedUsesOriginalSelectorSnapshot(t *testing.T) {
	input := []byte(`module "tfe_workspace" "example2" {
  name = "example-rtl-int-workspace2-prj"
}
`)

	edits := []Edit{
		SearchReplaceEdit{Old: "example2", New: "example2prd"},
		InsertHCLEdit{
			HCL: `description = "managed by hcl-forge"`,
			TargetBlock: &BlockSelector{
				Type:   "module",
				Labels: []string{"tfe_workspace", "example2"},
			},
		},
	}

	updated, results, err := ApplyEdits(input, edits)
	if err != nil {
		t.Fatalf("apply edits: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 edit results, got %d", len(results))
	}

	out := string(updated)
	if !strings.Contains(out, `module "tfe_workspace" "example2prd"`) {
		t.Fatalf("expected first edit to rename module label, got:\n%s", out)
	}

	if !strings.Contains(out, `description = "managed by hcl-forge"`) {
		t.Fatalf("expected insert to target old selector snapshot, got:\n%s", out)
	}
}

func TestInsertHCLEdit_PlacementAppend_InsertsBeforeClosingBrace(t *testing.T) {
	input := `module "tfe_workspace" "example1" {
  execution_mode = "remote"
}
`

	edit := InsertHCLEdit{
		HCL: `lifecycle {
  prevent_destroy = true
}`,
		TargetBlock: &BlockSelector{
			Type:   "module",
			Labels: []string{"tfe_workspace", "example1"},
		},
		Placement: &InsertPlacement{Mode: "append"},
	}

	updated, result, err := edit.Apply([]byte(input))
	if err != nil {
		t.Fatalf("apply insert_hcl placement append: %v", err)
	}
	if !result.Changed {
		t.Fatalf("expected change")
	}

	out := string(updated)
	idxLifecycle := strings.Index(out, "lifecycle {")
	idxClosing := strings.LastIndex(out, "}")
	if idxLifecycle == -1 || idxClosing == -1 {
		t.Fatalf("expected lifecycle block and closing brace, got:\n%s", out)
	}
	if idxLifecycle > idxClosing {
		t.Fatalf("expected lifecycle inserted before closing brace, got:\n%s", out)
	}
}

func TestInsertHCLEdit_PlacementAfterAttribute_InsertsAtAnchor(t *testing.T) {
	input := `module "tfe_workspace" "example1" {
  name           = "example1"
  execution_mode = "remote"
  tag_names      = ["hcl-forge"]
}
`

	edit := InsertHCLEdit{
		HCL: `lifecycle {
  prevent_destroy = true
}`,
		TargetBlock: &BlockSelector{
			Type:   "module",
			Labels: []string{"tfe_workspace", "example1"},
		},
		Placement: &InsertPlacement{Mode: "after_attribute", Attribute: "execution_mode", Strict: true},
	}

	updated, result, err := edit.Apply([]byte(input))
	if err != nil {
		t.Fatalf("apply insert_hcl placement after_attribute: %v", err)
	}
	if !result.Changed {
		t.Fatalf("expected change")
	}

	out := string(updated)
	idxExec := strings.Index(out, "execution_mode = \"remote\"")
	idxLifecycle := strings.Index(out, "lifecycle {")
	idxTags := strings.Index(out, "tag_names")
	if idxExec == -1 || idxLifecycle == -1 || idxTags == -1 {
		t.Fatalf("expected execution_mode, lifecycle and tag_names, got:\n%s", out)
	}
	if idxExec >= idxLifecycle || idxLifecycle >= idxTags {
		t.Fatalf("expected lifecycle inserted after execution_mode and before tag_names, got:\n%s", out)
	}
}
