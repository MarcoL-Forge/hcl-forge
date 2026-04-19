package cli

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// CLI render tests:
// - TestRunRenderWritesOutputFile verifies that `render --out` writes the rendered Terraform to the requested file path.
// - TestRunRenderWritesToStdout verifies that `render --stdout` emits the rendered Terraform to standard output.
// - writeTempTerraformFile creates a temporary Terraform input file for each test case.
// - captureFileOutput captures stdout or stderr so command output can be asserted without spawning a subprocess.

func TestRunRenderWritesOutputFile(t *testing.T) {
	inputPath := writeTempTerraformFile(t, "resource \"null_resource\" \"example\" {}\n")
	outputPath := filepath.Join(t.TempDir(), "rendered", "output.tf")

	stderrOutput := captureFileOutput(t, os.Stderr, func() {
		if err := Run([]string{"render", "--in", inputPath, "--out", outputPath}); err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	})

	renderedRaw, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read output file: %v", err)
	}

	if got, want := string(renderedRaw), "resource \"null_resource\" \"example\" {}\n"; got != want {
		t.Fatalf("unexpected rendered output: got %q want %q", got, want)
	}

	if !strings.Contains(stderrOutput, outputPath) {
		t.Fatalf("stderr did not mention output path: %q", stderrOutput)
	}
}

func TestRunRenderWritesToStdout(t *testing.T) {
	inputPath := writeTempTerraformFile(t, "locals {\n  region = \"us-central1\"\n}\n")

	stdoutOutput := captureFileOutput(t, os.Stdout, func() {
		if err := Run([]string{"render", "--in", inputPath, "--stdout"}); err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	})

	if got, want := stdoutOutput, "locals {\n  region = \"us-central1\"\n}\n"; got != want {
		t.Fatalf("unexpected stdout render: got %q want %q", got, want)
	}
}

func writeTempTerraformFile(t *testing.T, contents string) string {
	t.Helper()

	filePath := filepath.Join(t.TempDir(), "input.tf")
	if err := os.WriteFile(filePath, []byte(contents), 0o600); err != nil {
		t.Fatalf("write temp terraform file: %v", err)
	}

	return filePath
}

func captureFileOutput(t *testing.T, target *os.File, fn func()) string {
	t.Helper()

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}

	var restore func()
	switch target {
	case os.Stdout:
		original := os.Stdout
		os.Stdout = writer
		restore = func() { os.Stdout = original }
	case os.Stderr:
		original := os.Stderr
		os.Stderr = writer
		restore = func() { os.Stderr = original }
	default:
		reader.Close()
		writer.Close()
		t.Fatal("unsupported target file for capture")
	}

	outputCh := make(chan string, 1)
	go func() {
		captured, _ := io.ReadAll(reader)
		outputCh <- string(captured)
	}()

	fn()

	writer.Close()
	if restore != nil {
		restore()
	}

	output := <-outputCh
	reader.Close()

	return output
}
