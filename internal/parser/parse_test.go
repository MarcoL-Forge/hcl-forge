package parser

import (
	"bytes"
	"testing"

	"github.com/Marc0l95/hcl-forge/internal/document"
)

// Parser replace tests:
// - TestReplaceAttributeValue verifies that a targeted block attribute can be replaced with an HCL2-backed update.
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
