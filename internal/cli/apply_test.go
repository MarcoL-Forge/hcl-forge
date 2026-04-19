package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// CLI apply tests:
// - TestRunApplyWritesOutputFile verifies that `apply --playbook` reads a YAML playbook and writes the transformed file.
// - TestRunApplyWritesToStdout verifies that stdout remains a CLI concern via the --stdout flag.
// - TestRunApplySelectorTarget verifies that playbooks can target nested Terraform attributes via a generic selector.

func TestRunApplyWritesOutputFile(t *testing.T) {
	workspaceDir := t.TempDir()
	inputPath := filepath.Join(workspaceDir, "input.tf")
	outputPath := filepath.Join(workspaceDir, "rendered", "output.tf")
	playbookPath := filepath.Join(workspaceDir, "playbook.yaml")

	if err := os.WriteFile(inputPath, []byte("module \"network\" {\n  source = \"./old\"\n}\n"), 0o600); err != nil {
		t.Fatalf("write input file: %v", err)
	}

	playbookRaw := []byte("version: 1\ninput: input.tf\noutput: rendered/output.tf\noperations:\n  - op: set_attribute\n    block_type: module\n    labels: [network]\n    attribute: source\n    value: ./new\n")
	if err := os.WriteFile(playbookPath, playbookRaw, 0o600); err != nil {
		t.Fatalf("write playbook: %v", err)
	}

	stderrOutput := captureFileOutput(t, os.Stderr, func() {
		if err := Run([]string{"apply", "--playbook", playbookPath}); err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	})

	renderedRaw, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}

	if !strings.Contains(string(renderedRaw), "source = \"./new\"") {
		t.Fatalf("output did not contain replaced attribute: %q", renderedRaw)
	}

	if !strings.Contains(stderrOutput, outputPath) {
		t.Fatalf("stderr did not mention output path: %q", stderrOutput)
	}
}

func TestRunApplyWritesToStdout(t *testing.T) {
	workspaceDir := t.TempDir()
	inputPath := filepath.Join(workspaceDir, "input.tf")
	playbookPath := filepath.Join(workspaceDir, "playbook.yaml")

	if err := os.WriteFile(inputPath, []byte("resource \"null_resource\" \"example\" {}\n"), 0o600); err != nil {
		t.Fatalf("write input file: %v", err)
	}

	playbookRaw := []byte("version: 1\ninput: input.tf\noperations:\n  - op: set_attribute\n    block_type: resource\n    labels: [null_resource, example]\n    attribute: name\n    value: replacement\n")
	if err := os.WriteFile(playbookPath, playbookRaw, 0o600); err != nil {
		t.Fatalf("write playbook: %v", err)
	}

	stdoutOutput := captureFileOutput(t, os.Stdout, func() {
		if err := Run([]string{"apply", "--playbook", playbookPath, "--stdout"}); err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	})

	if !strings.Contains(stdoutOutput, "name = \"replacement\"") {
		t.Fatalf("stdout did not contain replaced attribute: %q", stdoutOutput)
	}
}

func TestRunApplySelectorTarget(t *testing.T) {
	workspaceDir := t.TempDir()
	inputPath := filepath.Join(workspaceDir, "input.tf")
	outputPath := filepath.Join(workspaceDir, "output.tf")
	playbookPath := filepath.Join(workspaceDir, "playbook.yaml")

	if err := os.WriteFile(inputPath, []byte("resource \"google_container_cluster\" \"this\" {\n  node_config {\n    service_account = \"old@example.com\"\n  }\n}\n"), 0o600); err != nil {
		t.Fatalf("write input file: %v", err)
	}

	playbookRaw := []byte("version: 1\ninput: input.tf\noutput: output.tf\noperations:\n  - op: set_attribute\n    target: resource.google_container_cluster.this.node_config.service_account\n    value: new@example.com\n    value_type: string\n")
	if err := os.WriteFile(playbookPath, playbookRaw, 0o600); err != nil {
		t.Fatalf("write playbook: %v", err)
	}

	if err := Run([]string{"apply", "--playbook", playbookPath}); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	renderedRaw, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}

	if !strings.Contains(string(renderedRaw), "service_account = \"new@example.com\"") {
		t.Fatalf("output did not contain selector-based replacement: %q", renderedRaw)
	}
}
