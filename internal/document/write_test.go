package document

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteDocument(t *testing.T) {
	t.Parallel()

	outputPath := filepath.Join(t.TempDir(), "generated", "output.tf")
	expectedRaw := []byte("resource \"null_resource\" \"written\" {}\n")
	doc := &Document{
		Path: "input.tf",
		Raw:  expectedRaw,
	}

	if err := WriteDocument(doc, outputPath); err != nil {
		t.Fatalf("WriteDocument returned error: %v", err)
	}

	actualRaw, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read written file: %v", err)
	}

	if !bytes.Equal(actualRaw, expectedRaw) {
		t.Fatalf("unexpected written bytes: got %q want %q", actualRaw, expectedRaw)
	}
}

func TestWriteDocumentNilDocument(t *testing.T) {
	t.Parallel()

	err := WriteDocument(nil, filepath.Join(t.TempDir(), "output.tf"))
	if err == nil {
		t.Fatal("WriteDocument returned nil error for nil document")
	}
}

func TestWriteDocumentEmptyOutputPath(t *testing.T) {
	t.Parallel()

	err := WriteDocument(&Document{Path: "input.tf", Raw: []byte("test")}, "")
	if err == nil {
		t.Fatal("WriteDocument returned nil error for empty output path")
	}
}
