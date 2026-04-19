package playbook

import (
	"os"
	"path/filepath"
	"testing"
)

// Playbook load tests:
// - TestLoad verifies that YAML playbooks are parsed and relative input/output paths are resolved from the playbook directory.
// - TestLoadMissingOperations verifies that a playbook without operations is rejected.

func TestLoad(t *testing.T) {
	t.Parallel()

	playbookDir := t.TempDir()
	playbookPath := filepath.Join(playbookDir, "playbook.yaml")
	raw := []byte("version: 1\ninput: inputs/main.tf\noutput: outputs/result.tf\noperations:\n  - op: set_attribute\n    target: module.network.source\n    value: ./modules/network\n")

	if err := os.WriteFile(playbookPath, raw, 0o600); err != nil {
		t.Fatalf("write playbook: %v", err)
	}

	pb, err := Load(playbookPath)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}

	if got, want := pb.Input, filepath.Join(playbookDir, "inputs", "main.tf"); got != want {
		t.Fatalf("unexpected input path: got %q want %q", got, want)
	}

	if got, want := pb.Output, filepath.Join(playbookDir, "outputs", "result.tf"); got != want {
		t.Fatalf("unexpected output path: got %q want %q", got, want)
	}

	if got, want := pb.Operations[0].Target, "module.network.source"; got != want {
		t.Fatalf("unexpected operation target: got %q want %q", got, want)
	}
}

func TestLoadMissingOperations(t *testing.T) {
	t.Parallel()

	playbookPath := filepath.Join(t.TempDir(), "playbook.yaml")
	raw := []byte("version: 1\ninput: main.tf\n")

	if err := os.WriteFile(playbookPath, raw, 0o600); err != nil {
		t.Fatalf("write playbook: %v", err)
	}

	_, err := Load(playbookPath)
	if err == nil {
		t.Fatal("Load returned nil error for missing operations")
	}
}
