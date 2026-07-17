package logging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNew_UnsupportedLevel(t *testing.T) {
	_, _, err := New(Config{Level: "trace", Format: "text", Output: "stderr"})
	if err == nil {
		t.Fatalf("expected error for unsupported log level")
	}
}

func TestNew_UnsupportedFormat(t *testing.T) {
	_, _, err := New(Config{Level: "info", Format: "xml", Output: "stderr"})
	if err == nil {
		t.Fatalf("expected error for unsupported log format")
	}
}

func TestNew_FileOutput(t *testing.T) {
	tmp := t.TempDir()
	logPath := filepath.Join(tmp, "hclforge.log")

	logger, closer, err := New(Config{Level: "info", Format: "text", Output: logPath})
	if err != nil {
		t.Fatalf("create logger: %v", err)
	}

	logger.Info("test_event", map[string]any{"k": "v"})

	if closer != nil {
		if err := closer.Close(); err != nil {
			t.Fatalf("close logger output: %v", err)
		}
	}

	data, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}

	out := string(data)
	if !strings.Contains(out, "msg=test_event") {
		t.Fatalf("expected text log output, got: %q", out)
	}
}

func TestNew_ArtifactOutput(t *testing.T) {
	tmp := t.TempDir()
	logPath := filepath.Join(tmp, "hclforge.log")
	artifactPath := filepath.Join(tmp, "hclforge.ndjson")

	logger, closer, err := New(Config{Level: "info", Format: "text", Output: logPath, Artifact: artifactPath})
	if err != nil {
		t.Fatalf("create logger: %v", err)
	}

	logger.Info("artifact_event", map[string]any{"k": "v"})

	if closer != nil {
		if err := closer.Close(); err != nil {
			t.Fatalf("close logger output: %v", err)
		}
	}

	data, err := os.ReadFile(artifactPath)
	if err != nil {
		t.Fatalf("read artifact file: %v", err)
	}

	out := string(data)
	if !strings.Contains(out, "artifact_event") {
		t.Fatalf("expected artifact event in ndjson output, got: %q", out)
	}
}

func TestDefaultLogger_SetAndGet(t *testing.T) {
	logger, closer, err := New(Config{Level: "info", Format: "text", Output: "stderr"})
	if err != nil {
		t.Fatalf("create logger: %v", err)
	}
	if closer != nil {
		defer func() {
			_ = closer.Close()
		}()
	}

	SetDefault(logger)
	if Default() != logger {
		t.Fatalf("expected default logger to match set logger")
	}
}

func TestLogger_AddsSchemaAndEventID(t *testing.T) {
	tmp := t.TempDir()
	artifactPath := filepath.Join(tmp, "events.ndjson")

	logger, closer, err := New(Config{Level: "info", Format: "json", Output: "stderr", Artifact: artifactPath})
	if err != nil {
		t.Fatalf("create logger: %v", err)
	}

	logger.Info("event_a", map[string]any{"k": "v"})
	logger.Info("event_b", map[string]any{"k": "v2"})

	if closer != nil {
		if err := closer.Close(); err != nil {
			t.Fatalf("close logger output: %v", err)
		}
	}

	data, err := os.ReadFile(artifactPath)
	if err != nil {
		t.Fatalf("read artifact file: %v", err)
	}

	out := string(data)
	if !strings.Contains(out, "\"schema_version\":\"hclforge.log.v1\"") {
		t.Fatalf("expected schema_version in output, got: %q", out)
	}
	if !strings.Contains(out, "\"event_id\":1") || !strings.Contains(out, "\"event_id\":2") {
		t.Fatalf("expected incrementing event_id values, got: %q", out)
	}
}

func TestLogger_RedactsSensitiveFields(t *testing.T) {
	tmp := t.TempDir()
	artifactPath := filepath.Join(tmp, "events.ndjson")

	logger, closer, err := New(Config{
		Level:      "info",
		Format:     "json",
		Output:     "stderr",
		Artifact:   artifactPath,
		RedactKeys: "custom_secret",
	})
	if err != nil {
		t.Fatalf("create logger: %v", err)
	}

	logger.Info("redaction_test", map[string]any{
		"token":         "abc123",
		"custom_secret": "shh",
		"nested": map[string]any{
			"password": "p@ss",
		},
	})

	if closer != nil {
		if err := closer.Close(); err != nil {
			t.Fatalf("close logger output: %v", err)
		}
	}

	data, err := os.ReadFile(artifactPath)
	if err != nil {
		t.Fatalf("read artifact file: %v", err)
	}

	out := string(data)
	if strings.Contains(out, "abc123") || strings.Contains(out, "shh") || strings.Contains(out, "p@ss") {
		t.Fatalf("expected sensitive values to be redacted, got: %q", out)
	}
	if !strings.Contains(out, "[REDACTED]") {
		t.Fatalf("expected redaction marker in output, got: %q", out)
	}
}
