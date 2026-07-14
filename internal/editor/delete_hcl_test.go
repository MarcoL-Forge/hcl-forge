package editor

import (
	"strings"
	"testing"
)

func TestDeleteHCLEdit_DeleteAttributeInsideBlock(t *testing.T) {
	input := `resource "google_storage_bucket" "bucket" {
  name     = "my-bucket"
  location = "europe-west1"
}
`

	edit := DeleteHCLEdit{
		TargetBlock: &BlockSelector{Type: "resource", Labels: []string{"google_storage_bucket", "bucket"}},
		Attribute:   "location",
	}

	updated, result, err := edit.Apply([]byte(input))
	if err != nil {
		t.Fatalf("apply delete_hcl: %v", err)
	}
	if !result.Changed {
		t.Fatalf("expected changed result")
	}

	out := string(updated)
	if strings.Contains(out, "location =") {
		t.Fatalf("expected location attribute removed, got:\n%s", out)
	}
}

func TestDeleteHCLEdit_DeleteWholeBlock(t *testing.T) {
	input := `resource "google_storage_bucket" "bucket" {
  name = "my-bucket"
}

variable "project_id" {
  type = string
}
`

	edit := DeleteHCLEdit{
		TargetBlock: &BlockSelector{Type: "variable", Labels: []string{"project_id"}},
	}

	updated, result, err := edit.Apply([]byte(input))
	if err != nil {
		t.Fatalf("apply delete_hcl block: %v", err)
	}
	if !result.Changed {
		t.Fatalf("expected changed result")
	}

	out := string(updated)
	if strings.Contains(out, `variable "project_id"`) {
		t.Fatalf("expected variable block removed, got:\n%s", out)
	}
}

func TestDeleteHCLEdit_RequiresTarget(t *testing.T) {
	input := `resource "google_storage_bucket" "bucket" {}`

	edit := DeleteHCLEdit{}
	_, _, err := edit.Apply([]byte(input))
	if err == nil {
		t.Fatalf("expected error for missing attribute/block selector")
	}
}

func TestDeleteHCLEdit_DeleteAllMatchingBlocks(t *testing.T) {
	input := `variable "a" {
  type = string
}

resource "google_storage_bucket" "bucket" {
  lifecycle {
    prevent_destroy = true
  }

  lifecycle {
    prevent_destroy = false
  }
}
`

	edit := DeleteHCLEdit{
		TargetBlock: &BlockSelector{Type: "lifecycle", Labels: []string{}},
		DeleteAll:   true,
	}

	updated, result, err := edit.Apply([]byte(input))
	if err != nil {
		t.Fatalf("apply delete_hcl all blocks: %v", err)
	}
	if !result.Changed || result.Occurrences != 2 {
		t.Fatalf("expected two deleted blocks, got changed=%v occurrences=%d", result.Changed, result.Occurrences)
	}

	out := string(updated)
	if strings.Contains(out, "lifecycle {") {
		t.Fatalf("expected all lifecycle blocks removed, got:\n%s", out)
	}
}

func TestDeleteHCLEdit_DeleteAllAttributesAcrossFile(t *testing.T) {
	input := `terraform {
  required_version = ">= 1.5.0"
}

resource "google_storage_bucket" "bucket" {
  name     = "my-bucket"
  location = "europe-west1"

  versioning {
    enabled = true
  }
}

resource "google_storage_bucket" "bucket2" {
  name     = "my-bucket-2"
  location = "us-central1"
}
`

	edit := DeleteHCLEdit{
		Attribute: "location",
		DeleteAll: true,
	}

	updated, result, err := edit.Apply([]byte(input))
	if err != nil {
		t.Fatalf("apply delete_hcl all attributes: %v", err)
	}
	if !result.Changed || result.Occurrences != 2 {
		t.Fatalf("expected two deleted attributes, got changed=%v occurrences=%d", result.Changed, result.Occurrences)
	}

	out := string(updated)
	if strings.Contains(out, "location =") {
		t.Fatalf("expected all location attributes removed, got:\n%s", out)
	}
}

func TestDeleteHCLEdit_DeleteDeepNestedBlockWithParents(t *testing.T) {
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

	edit := DeleteHCLEdit{
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
		t.Fatalf("apply delete_hcl nested block: %v", err)
	}
	if !result.Changed {
		t.Fatalf("expected changed result")
	}

	out := string(updated)
	if strings.Contains(out, "shielded_instance_config {") {
		t.Fatalf("expected nested block removed, got:\n%s", out)
	}
}

func TestDeleteHCLEdit_DeleteAttributeMissingTarget_IsNoOp(t *testing.T) {
	input := `resource "google_storage_bucket" "bucket" {
  name = "my-bucket"
}

`

	edit := DeleteHCLEdit{
		TargetBlock: &BlockSelector{Type: "resource", Labels: []string{"google_storage_bucket", "missing"}},
		Attribute:   "name",
	}

	updated, result, err := edit.Apply([]byte(input))
	if err != nil {
		t.Fatalf("expected no error when target block is missing, got: %v", err)
	}
	if result.Changed {
		t.Fatalf("expected no change for missing target")
	}
	if !strings.Contains(result.Message, "target block not found") {
		t.Fatalf("unexpected result message: %q", result.Message)
	}
	if string(updated) != input {
		t.Fatalf("expected input to remain unchanged")
	}
}

func TestDeleteHCLEdit_DeleteAllMatchingBlocks_WithWildcardLabel(t *testing.T) {
	input := `module "service-account-123" {
  source = "./modules/service-account"
}

module "service-account-456" {
  source = "./modules/service-account"
}

module "network" {
  source = "./modules/network"
}
`

	edit := DeleteHCLEdit{
		TargetBlock: &BlockSelector{Type: "module", Labels: []string{"service-account-*"}},
		DeleteAll:   true,
	}

	updated, result, err := edit.Apply([]byte(input))
	if err != nil {
		t.Fatalf("apply delete_hcl wildcard blocks: %v", err)
	}
	if !result.Changed || result.Occurrences != 2 {
		t.Fatalf("expected two deleted module blocks, got changed=%v occurrences=%d", result.Changed, result.Occurrences)
	}

	out := string(updated)
	if strings.Contains(out, `module "service-account-123"`) || strings.Contains(out, `module "service-account-456"`) {
		t.Fatalf("expected wildcard-matched modules removed, got:\n%s", out)
	}
	if !strings.Contains(out, `module "network"`) {
		t.Fatalf("expected non-matching module to remain, got:\n%s", out)
	}
}

func TestDeleteHCLEdit_DeleteAllAttributes_WithWildcardName(t *testing.T) {
	input := `resource "google_storage_bucket" "bucket" {
  enable_secure_boot = true
  enable_autoupgrade = false
  location           = "us-central1"
}
`

	edit := DeleteHCLEdit{
		Attribute: "enable_*",
		DeleteAll: true,
	}

	updated, result, err := edit.Apply([]byte(input))
	if err != nil {
		t.Fatalf("apply delete_hcl wildcard attributes: %v", err)
	}
	if !result.Changed || result.Occurrences != 2 {
		t.Fatalf("expected two deleted attributes, got changed=%v occurrences=%d", result.Changed, result.Occurrences)
	}

	out := string(updated)
	if strings.Contains(out, "enable_secure_boot") || strings.Contains(out, "enable_autoupgrade") {
		t.Fatalf("expected wildcard-matched attributes removed, got:\n%s", out)
	}
	if !strings.Contains(out, "location") {
		t.Fatalf("expected non-matching attribute to remain, got:\n%s", out)
	}
}

func TestDeleteHCLEdit_KeepOnly_WithGlobPattern(t *testing.T) {
	input := `resource "tfe_workspace" "example1" {
  name = "example-1"
}

resource "tfe_workspace" "example2" {
  name = "example-2"
}

resource "tfe_workspace" "team-dev" {
  name = "team-dev"
}

resource "google_storage_bucket" "logs" {
  name = "logs"
}
`

	edit := DeleteHCLEdit{
		KeepOnly: true,
		TargetBlock: &BlockSelector{
			Type:   "resource",
			Labels: []string{"tfe_workspace", "example*"},
		},
		MatchMode: "glob",
	}

	updated, result, err := edit.Apply([]byte(input))
	if err != nil {
		t.Fatalf("apply delete_hcl keep_only glob: %v", err)
	}
	if !result.Changed || result.Occurrences != 1 {
		t.Fatalf("expected one removed block, got changed=%v occurrences=%d", result.Changed, result.Occurrences)
	}

	out := string(updated)
	if !strings.Contains(out, `resource "tfe_workspace" "example1"`) || !strings.Contains(out, `resource "tfe_workspace" "example2"`) {
		t.Fatalf("expected selected workspaces to remain, got:\n%s", out)
	}
	if strings.Contains(out, `resource "tfe_workspace" "team-dev"`) {
		t.Fatalf("expected non-selected workspace removed, got:\n%s", out)
	}
	if !strings.Contains(out, `resource "google_storage_bucket" "logs"`) {
		t.Fatalf("expected non-scope resource to remain, got:\n%s", out)
	}
}

func TestDeleteHCLEdit_KeepOnly_WithRegexPattern(t *testing.T) {
	input := `resource "tfe_workspace" "example1" {
  name = "example-1"
}

resource "tfe_workspace" "example2" {
  name = "example-2"
}

resource "tfe_workspace" "example3" {
  name = "example-3"
}
`

	edit := DeleteHCLEdit{
		KeepOnly: true,
		TargetBlock: &BlockSelector{
			Type:   "resource",
			Labels: []string{"tfe_workspace", "example(1|3)"},
		},
		MatchMode: "regex",
	}

	updated, result, err := edit.Apply([]byte(input))
	if err != nil {
		t.Fatalf("apply delete_hcl keep_only regex: %v", err)
	}
	if !result.Changed || result.Occurrences != 1 {
		t.Fatalf("expected one removed block, got changed=%v occurrences=%d", result.Changed, result.Occurrences)
	}

	out := string(updated)
	if !strings.Contains(out, `resource "tfe_workspace" "example1"`) || !strings.Contains(out, `resource "tfe_workspace" "example3"`) {
		t.Fatalf("expected regex-selected workspaces to remain, got:\n%s", out)
	}
	if strings.Contains(out, `resource "tfe_workspace" "example2"`) {
		t.Fatalf("expected regex non-selected workspace removed, got:\n%s", out)
	}
}

func TestDeleteHCLEdit_KeepOnly_RequiresBlock(t *testing.T) {
	input := `resource "tfe_workspace" "example1" {
  name = "example-1"
}
`

	edit := DeleteHCLEdit{KeepOnly: true}

	_, _, err := edit.Apply([]byte(input))
	if err == nil {
		t.Fatalf("expected error for keep_only without block")
	}
}
