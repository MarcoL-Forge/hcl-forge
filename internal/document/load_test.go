package document

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// Document load tests:
// - TestLoadDocument verifies that a Terraform file can be loaded from disk and its bytes are preserved exactly.
// - TestLoadDocumentMissingFile verifies that loading a missing file returns an error and no partial document.

// TODO: Add table-driven coverage for loader edge cases:
// - empty files
// - very large files
// - unreadable files or permission errors
// - directories passed to LoadDocument
// - non-HCL/Terraform file extensions if extension filtering is added
// - paths containing nested directories or unusual characters

func TestLoadDocument(t *testing.T) {
	t.Parallel()

	// Create a real Terraform file so the loader is exercised through the filesystem.
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "main.tf")
	expectedRaw := []byte("resource \"null_resource\" \"example\" {}\n")

	if err := os.WriteFile(filePath, expectedRaw, 0o600); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	doc, err := LoadDocument(filePath)
	if err != nil {
		t.Fatalf("LoadDocument returned error: %v", err)
	}

	if doc == nil {
		t.Fatal("LoadDocument returned nil document")
	}

	if doc.Path != filePath {
		t.Fatalf("unexpected path: got %q want %q", doc.Path, filePath)
	}

	// The loader should preserve the original file bytes exactly as written.
	if !bytes.Equal(doc.Raw, expectedRaw) {
		t.Fatalf("unexpected raw bytes: got %q want %q", doc.Raw, expectedRaw)
	}
}

func TestLoadDocumentMissingFile(t *testing.T) {
	t.Parallel()

	// A missing file should fail cleanly and never return a partial document.
	missingPath := filepath.Join(t.TempDir(), "missing.tf")

	doc, err := LoadDocument(missingPath)
	if err == nil {
		t.Fatal("LoadDocument returned nil error for missing file")
	}

	if doc != nil {
		t.Fatalf("LoadDocument returned document for missing file: %+v", doc)
	}
}
