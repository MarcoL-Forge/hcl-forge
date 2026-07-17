package logging

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

type Config struct {
	Verbose    bool
	Level      string
	Format     string
	Output     string
	Artifact   string
	RedactKeys string
}

type Logger struct {
	mu            sync.Mutex
	level         Level
	format        string
	out           io.Writer
	artifact      io.Writer
	sequence      uint64
	schemaVersion string
	redactedKeys  map[string]struct{}
}

var defaultLogger atomic.Pointer[Logger]

func New(cfg Config) (*Logger, io.Closer, error) {
	level, err := parseLevel(cfg.Level)
	if err != nil {
		return nil, nil, err
	}

	if cfg.Verbose {
		level = LevelDebug
	}

	format := strings.ToLower(strings.TrimSpace(cfg.Format))
	if format == "" {
		format = "text"
	}
	if format != "text" && format != "json" {
		return nil, nil, fmt.Errorf("unsupported log format %q", cfg.Format)
	}

	output := strings.TrimSpace(cfg.Output)
	if output == "" {
		output = "stderr"
	}

	var out io.Writer
	closers := make([]io.Closer, 0, 2)

	switch output {
	case "stderr":
		out = os.Stderr
	case "stdout":
		out = os.Stdout
	default:
		f, err := os.OpenFile(output, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return nil, nil, fmt.Errorf("open log output %q: %w", output, err)
		}
		out = f
		closers = append(closers, f)
	}

	var artifact io.Writer
	artifactPath := strings.TrimSpace(cfg.Artifact)
	if artifactPath != "" {
		f, err := os.OpenFile(artifactPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			for _, c := range closers {
				_ = c.Close()
			}
			return nil, nil, fmt.Errorf("open log artifact %q: %w", artifactPath, err)
		}
		artifact = f
		closers = append(closers, f)
	}

	redactedKeys := defaultRedactedKeys()
	for _, key := range parseCSV(cfg.RedactKeys) {
		redactedKeys[key] = struct{}{}
	}

	return &Logger{
		level:         level,
		format:        format,
		out:           out,
		artifact:      artifact,
		schemaVersion: "hclforge.log.v1",
		redactedKeys:  redactedKeys,
	}, multiCloser(closers), nil
}

func (l *Logger) Debug(msg string, fields map[string]any) {
	l.log(LevelDebug, msg, fields)
}

func (l *Logger) Info(msg string, fields map[string]any) {
	l.log(LevelInfo, msg, fields)
}

func (l *Logger) Warn(msg string, fields map[string]any) {
	l.log(LevelWarn, msg, fields)
}

func (l *Logger) Error(msg string, fields map[string]any) {
	l.log(LevelError, msg, fields)
}

func (l *Logger) log(level Level, msg string, fields map[string]any) {
	if l == nil || level < l.level {
		return
	}

	if fields == nil {
		fields = map[string]any{}
	}

	fields = cloneFields(fields)
	fields = redactFields(fields, l.redactedKeys)
	fields["time"] = time.Now().UTC().Format(time.RFC3339)
	fields["level"] = level.String()
	fields["msg"] = msg
	fields["schema_version"] = l.schemaVersion
	fields["event_id"] = atomic.AddUint64(&l.sequence, 1)

	l.mu.Lock()
	defer l.mu.Unlock()

	switch l.format {
	case "json":
		data, err := json.Marshal(fields)
		if err != nil {
			if _, writeErr := fmt.Fprintf(l.out, "{\"level\":\"error\",\"msg\":\"log marshal failed\",\"error\":%q}\n", err.Error()); writeErr != nil {
				return
			}
			return
		}
		if _, writeErr := fmt.Fprintln(l.out, string(data)); writeErr != nil {
			return
		}
		if l.artifact != nil {
			if _, writeErr := fmt.Fprintln(l.artifact, string(data)); writeErr != nil {
				return
			}
		}
	default:
		if _, writeErr := fmt.Fprintln(l.out, formatText(fields)); writeErr != nil {
			return
		}
		if l.artifact != nil {
			data, err := json.Marshal(fields)
			if err == nil {
				if _, writeErr := fmt.Fprintln(l.artifact, string(data)); writeErr != nil {
					return
				}
			}
		}
	}
}

func SetDefault(logger *Logger) {
	defaultLogger.Store(logger)
}

func Default() *Logger {
	if logger := defaultLogger.Load(); logger != nil {
		return logger
	}

	logger, _, err := New(Config{Level: "error", Format: "text", Output: "stderr"})
	if err != nil {
		return nil
	}

	defaultLogger.Store(logger)
	return logger
}

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "debug"
	case LevelInfo:
		return "info"
	case LevelWarn:
		return "warn"
	case LevelError:
		return "error"
	default:
		return "unknown"
	}
}

func parseLevel(v string) (Level, error) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "info":
		return LevelInfo, nil
	case "debug":
		return LevelDebug, nil
	case "warn":
		return LevelWarn, nil
	case "error":
		return LevelError, nil
	default:
		return LevelInfo, fmt.Errorf("unsupported log level %q", v)
	}
}

func cloneFields(in map[string]any) map[string]any {
	out := make(map[string]any, len(in)+3)
	for k, v := range in {
		out[k] = v
	}
	return out
}

func redactFields(in map[string]any, redactedKeys map[string]struct{}) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		if _, redact := redactedKeys[strings.ToLower(k)]; redact {
			out[k] = "[REDACTED]"
			continue
		}

		out[k] = redactValue(v, redactedKeys)
	}

	return out
}

func redactValue(v any, redactedKeys map[string]struct{}) any {
	switch val := v.(type) {
	case map[string]any:
		return redactFields(val, redactedKeys)
	case []any:
		out := make([]any, len(val))
		for i := range val {
			out[i] = redactValue(val[i], redactedKeys)
		}
		return out
	default:
		return v
	}
}

func parseCSV(in string) []string {
	parts := strings.Split(in, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		v := strings.ToLower(strings.TrimSpace(part))
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

func defaultRedactedKeys() map[string]struct{} {
	keys := []string{
		"password",
		"passphrase",
		"secret",
		"token",
		"api_key",
		"apikey",
		"private_key",
		"client_secret",
		"credentials",
		"authorization",
	}

	out := make(map[string]struct{}, len(keys))
	for _, key := range keys {
		out[key] = struct{}{}
	}
	return out
}

func formatText(fields map[string]any) string {
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%v", k, fields[k]))
	}

	return strings.Join(parts, " ")
}

type closerFunc func() error

func (f closerFunc) Close() error {
	return f()
}

func multiCloser(closers []io.Closer) io.Closer {
	if len(closers) == 0 {
		return nil
	}

	return closerFunc(func() error {
		var firstErr error
		for _, c := range closers {
			if err := c.Close(); err != nil && firstErr == nil {
				firstErr = err
			}
		}
		return firstErr
	})
}
