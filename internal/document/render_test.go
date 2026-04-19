package document

import (
	"bytes"
	"testing"
)

func TestRenderDocument(t *testing.T) {
	t.Parallel()

	expectedRaw := []byte("resource \"null_resource\" \"rendered\" {}\n")
	doc := &Document{
		Path: "input.tf",
		Raw:  expectedRaw,
	}

	renderedRaw, err := RenderDocument(doc)
	if err != nil {
		t.Fatalf("RenderDocument returned error: %v", err)
	}

	if !bytes.Equal(renderedRaw, expectedRaw) {
		t.Fatalf("unexpected rendered bytes: got %q want %q", renderedRaw, expectedRaw)
	}
}

func TestRenderDocumentNilDocument(t *testing.T) {
	t.Parallel()

	_, err := RenderDocument(nil)
	if err == nil {
		t.Fatal("RenderDocument returned nil error for nil document")
	}
}
