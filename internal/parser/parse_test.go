package parser

import (
	"bytes"
	"testing"

	"github.com/Marc0l95/hcl-forge/internal/document"
)

// Parser replace tests:
// - TestReplaceAttributeValue verifies that a targeted block attribute can be replaced with an HCL2-backed update.
// - TestReplaceAttributeValueTopLevel verifies that a top-level tfvars-style attribute can be replaced without a block selector.
// - TestReplaceAttributeValueSelectorNestedBlock verifies that selector-based targeting can update nested block attributes.
// - TestReplaceAttributeValueSelectorHCL verifies that selector-based targeting can write raw HCL expressions.
// - TestReplaceAttributeValueMissingBlock verifies that targeting a missing block returns an error.

func TestReplaceAttributeValue(t *testing.T) {
	t.Parallel()

	doc := &document.Document{
		Path: "main.tf",
		Raw:  []byte("resource \"null_resource\" \"example\" {\n  triggers = {\n    region = \"us-central1\"\n  }\n}\n"),
	}

	replacedDoc, err := ReplaceAttributeValue(doc, ReplaceAttributeInput{
		BlockType: "resource",
		Labels:    []string{"null_resource", "example"},
		Attribute: "name",
		Value:     "replacement",
		ValueType: "string",
	})
	if err != nil {
		t.Fatalf("ReplaceAttributeValue returned error: %v", err)
	}

	if !bytes.Contains(replacedDoc.Raw, []byte("name = \"replacement\"")) {
		t.Fatalf("updated document did not contain replacement attribute: %q", replacedDoc.Raw)
	}
}

func TestReplaceAttributeValueTopLevel(t *testing.T) {
	t.Parallel()

	doc := &document.Document{
		Path: "terraform.tfvars",
		Raw:  []byte("gcp_region = \"us-central1\"\nenvironment = \"dev\"\n"),
	}

	replacedDoc, err := ReplaceAttributeValue(doc, ReplaceAttributeInput{
		Attribute: "gcp_region",
		Value:     "europe-west1",
		ValueType: "string",
	})
	if err != nil {
		t.Fatalf("ReplaceAttributeValue returned error: %v", err)
	}

	if !bytes.Contains(replacedDoc.Raw, []byte("\"europe-west1\"")) {
		t.Fatalf("updated document did not contain replaced top-level attribute: %q", replacedDoc.Raw)
	}

	if !bytes.Contains(replacedDoc.Raw, []byte("environment")) || !bytes.Contains(replacedDoc.Raw, []byte("\"dev\"")) {
		t.Fatalf("updated document did not preserve unrelated top-level attributes: %q", replacedDoc.Raw)
	}
}

func TestReplaceAttributeValueSelectorNestedBlock(t *testing.T) {
	t.Parallel()

	doc := &document.Document{
		Path: "main.tf",
		Raw:  []byte("resource \"google_container_cluster\" \"this\" {\n  node_config {\n    service_account = \"old@example.com\"\n  }\n}\n"),
	}

	replacedDoc, err := ReplaceAttributeValue(doc, ReplaceAttributeInput{
		Selector:  "resource.google_container_cluster.this.node_config.service_account",
		Value:     "new@example.com",
		ValueType: "string",
	})
	if err != nil {
		t.Fatalf("ReplaceAttributeValue returned error: %v", err)
	}

	if !bytes.Contains(replacedDoc.Raw, []byte("service_account = \"new@example.com\"")) {
		t.Fatalf("updated document did not contain replaced nested attribute: %q", replacedDoc.Raw)
	}
}

func TestReplaceAttributeValueSelectorHCL(t *testing.T) {
	t.Parallel()

	doc := &document.Document{
		Path: "terraform.tfvars",
		Raw:  []byte("node_tags = [\"gke\"]\n"),
	}

	replacedDoc, err := ReplaceAttributeValue(doc, ReplaceAttributeInput{
		Selector:  "node_tags",
		Value:     "[\"gke\", \"shared\"]",
		ValueType: "hcl",
	})
	if err != nil {
		t.Fatalf("ReplaceAttributeValue returned error: %v", err)
	}

	if !bytes.Contains(replacedDoc.Raw, []byte("shared")) {
		t.Fatalf("updated document did not contain replaced HCL expression: %q", replacedDoc.Raw)
	}
}

func TestReplaceAttributeValueMissingBlock(t *testing.T) {
	t.Parallel()

	doc := &document.Document{
		Path: "main.tf",
		Raw:  []byte("resource \"null_resource\" \"example\" {}\n"),
	}

	_, err := ReplaceAttributeValue(doc, ReplaceAttributeInput{
		BlockType: "module",
		Labels:    []string{"network"},
		Attribute: "source",
		Value:     "./module",
		ValueType: "string",
	})
	if err == nil {
		t.Fatal("ReplaceAttributeValue returned nil error for missing block")
	}
}
