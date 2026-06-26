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
