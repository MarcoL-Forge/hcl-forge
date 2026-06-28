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
