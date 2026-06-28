package document

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWriteFile_CreatesDirectories(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "nested", "dir", "main.tf")
	content := []byte("hello")

	if err := WriteFile(path, content); err != nil {
		t.Fatalf("write file: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read written file: %v", err)
	}
	if string(got) != string(content) {
		t.Fatalf("written content mismatch: got %q want %q", string(got), string(content))
	}
}

func TestWriteToTargetDir_UsesSourceBasename(t *testing.T) {
	targetDir := t.TempDir()
	out, err := WriteToTargetDir(filepath.Join("any", "path", "vars.tf"), targetDir, []byte("x"))
	if err != nil {
		t.Fatalf("write to target dir: %v", err)
	}

	want := filepath.Join(targetDir, "vars.tf")
	if out != want {
		t.Fatalf("output path mismatch: got %q want %q", out, want)
	}

	if _, err := os.Stat(out); err != nil {
		t.Fatalf("expected output file at %q: %v", out, err)
	}
}

func TestWriteFile_OverwritesExistingFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "main.tf")

	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatalf("seed file: %v", err)
	}

	if err := WriteFile(path, []byte("new")); err != nil {
		t.Fatalf("write file: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}

	if string(got) != "new" {
		t.Fatalf("expected overwritten content %q, got %q", "new", string(got))
	}
}
