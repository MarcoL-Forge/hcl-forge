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
