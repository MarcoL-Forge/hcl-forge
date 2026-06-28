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
