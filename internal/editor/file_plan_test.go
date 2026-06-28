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
