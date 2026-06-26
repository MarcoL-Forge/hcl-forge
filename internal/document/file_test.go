package document

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePathFrom_Relative(t *testing.T) {
	root := t.TempDir()
	got, err := ResolvePathFrom(root, filepath.Join("nested", "main.tf"))
	if err != nil {
		t.Fatalf("resolve path: %v", err)
	}

	want := filepath.Join(root, "nested", "main.tf")
	if got != want {
		t.Fatalf("resolved path mismatch: got %q want %q", got, want)
	}
}

func TestLoadFileWithPath(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "main.tf")
	content := []byte("resource \"null_resource\" \"x\" {}\n")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write fixture file: %v", err)
	}

	gotData, gotPath, err := LoadFileWithPath(path)
	if err != nil {
		t.Fatalf("load file with path: %v", err)
	}

	if string(gotData) != string(content) {
		t.Fatalf("content mismatch: got %q want %q", string(gotData), string(content))
	}
	if gotPath != path {
		t.Fatalf("path mismatch: got %q want %q", gotPath, path)
	}
}

func TestLoadFile_NotFound(t *testing.T) {
	_, err := LoadFile(filepath.Join(t.TempDir(), "missing.tf"))
	if err == nil {
		t.Fatalf("expected error for missing file")
	}
}
