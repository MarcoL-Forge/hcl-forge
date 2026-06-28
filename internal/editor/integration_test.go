//go:build integration

package editor

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"os"
)

// Integration test: copy a real fixture file, run ApplyFilePlans with a
// SearchReplaceEdit and assert the output file is written and changed.
func TestApplyFilePlans_Integration(t *testing.T) {
	tmp := t.TempDir()

	src := filepath.Join(tmp, "project.tf")
	fixture := []byte(`resource "google_project_service" "compute" {
  service = "compute.googleapis.com"
}
`)
	if err := os.WriteFile(src, fixture, 0o644); err != nil {
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
