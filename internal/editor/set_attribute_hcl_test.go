package editor

import (
	"strings"
	"testing"
)

func TestSetAttributeHCLEdit_UpdatesExistingAttribute(t *testing.T) {
	input := `resource "google_storage_bucket" "bucket" {
  location = "europe-west1"
}
`

	edit := SetAttributeHCLEdit{
		TargetBlock: &BlockSelector{Type: "resource", Labels: []string{"google_storage_bucket", "bucket"}},
		Attribute:   "location",
		ValueHCL:    "\"us-central1\"",
	}

	updated, result, err := edit.Apply([]byte(input))
	if err != nil {
		t.Fatalf("apply set_attribute: %v", err)
	}
	if !result.Changed {
		t.Fatalf("expected changed result")
	}

	out := string(updated)
	if !strings.Contains(out, `location = "us-central1"`) {
		t.Fatalf("expected updated attribute value, got:\n%s", out)
	}
}

func TestSetAttributeHCLEdit_CreatesWhenMissing(t *testing.T) {
	input := `resource "google_storage_bucket" "bucket" {
  name = "my-bucket"
}
`

	edit := SetAttributeHCLEdit{
		TargetBlock:     &BlockSelector{Type: "resource", Labels: []string{"google_storage_bucket", "bucket"}},
		Attribute:       "force_destroy",
		ValueHCL:        "true",
		CreateIfMissing: true,
	}

	updated, result, err := edit.Apply([]byte(input))
	if err != nil {
		t.Fatalf("apply set_attribute: %v", err)
	}
	if !result.Changed {
		t.Fatalf("expected changed result")
	}

	out := string(updated)
	if !strings.Contains(out, "force_destroy = true") {
		t.Fatalf("expected created attribute, got:\n%s", out)
	}
}

func TestSetAttributeHCLEdit_RequiresExistingWhenConfigured(t *testing.T) {
	input := `resource "google_storage_bucket" "bucket" {
  name = "my-bucket"
}
`

	edit := SetAttributeHCLEdit{
		TargetBlock: &BlockSelector{Type: "resource", Labels: []string{"google_storage_bucket", "bucket"}},
		Attribute:   "force_destroy",
		ValueHCL:    "true",
	}

	updated, result, err := edit.Apply([]byte(input))
	if err != nil {
		t.Fatalf("apply set_attribute: %v", err)
	}
	if result.Changed {
		t.Fatalf("expected no change when create_if_missing is false")
	}
	if result.Message != "attribute not found" {
		t.Fatalf("unexpected result message: %q", result.Message)
	}
	if string(updated) != input {
		t.Fatalf("expected unchanged input")
	}
}

func TestSetAttributeHCLEdit_TargetMissing_IsNoOp(t *testing.T) {
	input := `resource "google_storage_bucket" "bucket" {
  name = "my-bucket"
}
`

	edit := SetAttributeHCLEdit{
		TargetBlock: &BlockSelector{Type: "resource", Labels: []string{"google_storage_bucket", "missing"}},
		Attribute:   "location",
		ValueHCL:    "\"us-central1\"",
	}

	updated, result, err := edit.Apply([]byte(input))
	if err != nil {
		t.Fatalf("expected no error when target block is missing, got: %v", err)
	}
	if result.Changed {
		t.Fatalf("expected no change")
	}
	if !strings.Contains(result.Message, "target block not found") {
		t.Fatalf("unexpected result message: %q", result.Message)
	}
	if string(updated) != input {
		t.Fatalf("expected unchanged input")
	}
}

func TestSetAttributeHCLEdit_AlreadySet_IsNoOp(t *testing.T) {
	input := `resource "google_storage_bucket" "bucket" {
  force_destroy = true
}
`

	edit := SetAttributeHCLEdit{
		TargetBlock: &BlockSelector{Type: "resource", Labels: []string{"google_storage_bucket", "bucket"}},
		Attribute:   "force_destroy",
		ValueHCL:    "true",
	}

	updated, result, err := edit.Apply([]byte(input))
	if err != nil {
		t.Fatalf("apply set_attribute: %v", err)
	}
	if result.Changed {
		t.Fatalf("expected no change when value already set")
	}
	if result.Message != "attribute already set" {
		t.Fatalf("unexpected result message: %q", result.Message)
	}
	if string(updated) != input {
		t.Fatalf("expected unchanged output")
	}
}

func TestApplyEdits_SetAttributeScopedUsesOriginalSelectorSnapshot(t *testing.T) {
	input := []byte(`module "tfe_workspace" "example2" {
  name = "example-rtl-int-workspace2-prj"
}
`)

	edits := []Edit{
		SearchReplaceEdit{Old: "example2", New: "example2prd"},
		SetAttributeHCLEdit{
			TargetBlock: &BlockSelector{
				Type:   "module",
				Labels: []string{"tfe_workspace", "example2"},
			},
			Attribute: "name",
			ValueHCL:  `"renamed-by-set-attribute"`,
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

	if !strings.Contains(out, `name = "renamed-by-set-attribute"`) {
		t.Fatalf("expected set_attribute to target old selector snapshot, got:\n%s", out)
	}
}
