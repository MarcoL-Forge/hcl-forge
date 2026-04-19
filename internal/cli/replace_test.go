package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// CLI replace tests:
// - TestRunReplaceWritesOutputFile verifies that `replace --out` updates a targeted block attribute and writes the result.
// - TestRunReplaceWritesToStdout verifies that `replace --stdout` emits the updated Terraform to standard output.
// - TestRunReplaceSelectorTarget verifies that `replace --target` can update nested Terraform attributes.

func TestRunReplaceWritesOutputFile(t *testing.T) {
	inputPath := writeTempTerraformFile(t, "resource \"null_resource\" \"example\" {}\n")
	outputPath := filepath.Join(t.TempDir(), "rendered", "output.tf")

	stderrOutput := captureFileOutput(t, os.Stderr, func() {
		err := Run([]string{
			"replace",
			"--in", inputPath,
			"--block-type", "resource",
			"--labels", "null_resource,example",
			"--attr", "name",
			"--value", "replacement",
			"--out", outputPath,
		})
		if err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	})

	renderedRaw, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}

	if !strings.Contains(string(renderedRaw), "name = \"replacement\"") {
		t.Fatalf("output did not contain replaced attribute: %q", renderedRaw)
	}

	if !strings.Contains(stderrOutput, outputPath) {
		t.Fatalf("stderr did not mention output path: %q", stderrOutput)
	}
}

func TestRunReplaceWritesToStdout(t *testing.T) {
	inputPath := writeTempTerraformFile(t, "module \"network\" {\n  source = \"./old\"\n}\n")

	stdoutOutput := captureFileOutput(t, os.Stdout, func() {
		err := Run([]string{
			"replace",
			"--in", inputPath,
			"--block-type", "module",
			"--labels", "network",
			"--attr", "source",
			"--value", "./new",
			"--stdout",
		})
		if err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	})

	if !strings.Contains(stdoutOutput, "source = \"./new\"") {
		t.Fatalf("stdout did not contain replaced attribute: %q", stdoutOutput)
	}
}

func TestRunReplaceSelectorTarget(t *testing.T) {
	inputPath := writeTempTerraformFile(t, "resource \"google_container_cluster\" \"this\" {\n  node_config {\n    service_account = \"old@example.com\"\n  }\n}\n")

	stdoutOutput := captureFileOutput(t, os.Stdout, func() {
		err := Run([]string{
			"replace",
			"--in", inputPath,
			"--target", "resource.google_container_cluster.this.node_config.service_account",
			"--value", "new@example.com",
			"--stdout",
		})
		if err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	})

	if !strings.Contains(stdoutOutput, "service_account = \"new@example.com\"") {
		t.Fatalf("stdout did not contain selector-based replacement: %q", stdoutOutput)
	}
}
