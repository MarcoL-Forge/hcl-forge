package editor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestApplyFilePlans_ErrorsOnDuplicateOutputPath(t *testing.T) {
	tmp := t.TempDir()

	srcA := filepath.Join(tmp, "a.tf")
	srcB := filepath.Join(tmp, "b.tf")
	if err := os.WriteFile(srcA, []byte("x = \"old\"\n"), 0o644); err != nil {
		t.Fatalf("write source A: %v", err)
	}
	if err := os.WriteFile(srcB, []byte("x = \"old\"\n"), 0o644); err != nil {
		t.Fatalf("write source B: %v", err)
	}

	out := filepath.Join(tmp, "out.tf")

	plans := []FilePlan{
		{
			SourcePath: srcA,
			OutputPath: out,
			Edits: []Edit{
				SearchReplaceEdit{Old: "old", New: "new"},
			},
		},
		{
			SourcePath: srcB,
			OutputPath: out,
			Edits: []Edit{
				SearchReplaceEdit{Old: "old", New: "new"},
			},
		},
	}

	_, err := ApplyFilePlans(plans, 2)
	if err == nil {
		t.Fatalf("expected duplicate output path error")
	}

	if !strings.Contains(err.Error(), "duplicate output path") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPlanFilePlans_CollectsPartialFailures(t *testing.T) {
	tmp := t.TempDir()

	okFile := filepath.Join(tmp, "ok.tf")
	if err := os.WriteFile(okFile, []byte("x = \"old\"\n"), 0o644); err != nil {
		t.Fatalf("write ok file: %v", err)
	}

	missingFile := filepath.Join(tmp, "missing.tf")

	plans := []FilePlan{
		{
			SourcePath: okFile,
			OutputPath: okFile,
			Edits:      []Edit{SearchReplaceEdit{Old: "old", New: "new"}},
		},
		{
			SourcePath: missingFile,
			OutputPath: missingFile,
			Edits:      []Edit{SearchReplaceEdit{Old: "old", New: "new"}},
		},
	}

	results, err := PlanFilePlans(plans, 2)
	if err == nil {
		t.Fatalf("expected aggregated error")
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	if !results[0].Changed {
		t.Fatalf("expected successful file to be planned as changed")
	}

	if results[1].Error == "" {
		t.Fatalf("expected failed file result to include error")
	}
}

func TestClampWorkers(t *testing.T) {
	tests := []struct {
		name    string
		workers int
		max     int
		want    int
	}{
		{name: "default when non-positive", workers: 0, max: 5, want: 4},
		{name: "cap by max", workers: 10, max: 5, want: 5},
		{name: "keep within range", workers: 3, max: 5, want: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := clampWorkers(tt.workers, tt.max)
			if got != tt.want {
				t.Fatalf("clampWorkers(%d,%d)=%d, want %d", tt.workers, tt.max, got, tt.want)
			}
		})
	}
}
