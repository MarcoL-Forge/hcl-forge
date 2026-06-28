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
