//go:build integration

package editor

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// Integration test: copy a real fixture file, run ApplyFilePlans with a
// SearchReplaceEdit and assert the output file is written and changed.
func TestApplyFilePlans_Integration(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working directory: %v", err)
	}

	// go test runs this package from internal/editor, so walk up to repo root.
	fixture := filepath.Clean(filepath.Join(cwd, "..", "..", "testing", "gke", "project.tf"))

	data, err := os.ReadFile(fixture)
	if err != nil {
		t.Fatalf("read fixture %q: %v", fixture, err)
	}

	tmp := t.TempDir()

	src := filepath.Join(tmp, "project.tf")
	if err := os.WriteFile(src, data, 0o644); err != nil {
		t.Fatalf("write temp source: %v", err)
	}

	out := filepath.Join(tmp, "project.out.tf")

	plans := []FilePlan{
		{
			SourcePath: src,
			OutputPath: out,
			Edits: []Edit{
				SearchReplaceEdit{Old: "compute.googleapis.com", New: "compute.googleapis.com-REPLACED"},
			},
		},
	}

	results, err := ApplyFilePlans(plans, 1)
	if err != nil {
		t.Fatalf("apply file plans: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	res := results[0]
	if !res.Changed {
		t.Fatalf("expected change to be true, results: %+v", res.Results)
	}

	outData, err := os.ReadFile(res.OutputPath)
	if err != nil {
		t.Fatalf("read output file %q: %v", res.OutputPath, err)
	}

	if !bytes.Contains(outData, []byte("compute.googleapis.com-REPLACED")) {
		t.Fatalf("output file did not contain replacement")
	}
}
