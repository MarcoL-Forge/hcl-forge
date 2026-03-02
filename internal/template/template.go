// Package template provides a Go text/template evaluator for hclforge
// transformation expressions, exposing .Vars and .Flags to templates.
package template

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

// Context is the data available inside template expressions.
type Context struct {
	// Vars holds string values from the spec's vars section.
	Vars map[string]string

	// Flags holds boolean values from the spec's flags section.
	Flags map[string]bool
}

// Evaluator renders Go text/template expressions against a fixed Context.
type Evaluator struct {
	ctx  Context
	tmpl *template.Template
}

// New returns an Evaluator pre-loaded with vars and flags.
func New(vars map[string]string, flags map[string]bool) *Evaluator {
	return &Evaluator{
		ctx: Context{Vars: vars, Flags: flags},
	}
}

// Render evaluates a template string and returns the result.
// Strings that contain no template markers are returned unchanged.
func (e *Evaluator) Render(input string) (string, error) {
	if !strings.Contains(input, "{{") {
		return input, nil
	}

	tmpl, err := template.New("").Funcs(funcMap()).Option("missingkey=zero").Parse(input)
	if err != nil {
		return "", fmt.Errorf("parsing template %q: %w", input, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, e.ctx); err != nil {
		return "", fmt.Errorf("executing template %q: %w", input, err)
	}

	return buf.String(), nil
}

// EvalCondition evaluates an `if` expression and returns true/false.
// An empty condition is treated as unconditionally true.
// Supports both bare expressions (.Flags.x) and wrapped ones ({{ .Flags.x }}).
func (e *Evaluator) EvalCondition(expr string) (bool, error) {
	if expr == "" {
		return true, nil
	}

	inner := unwrap(expr)
	wrapped := fmt.Sprintf(`{{ if %s }}true{{ else }}false{{ end }}`, inner)

	result, err := e.Render(wrapped)
	if err != nil {
		return false, fmt.Errorf("evaluating condition %q: %w", expr, err)
	}

	return strings.TrimSpace(result) == "true", nil
}

// unwrap strips outer {{ and }} delimiters from an expression if present.
// This allows callers to pass either "{{ .Flags.x }}" or ".Flags.x".
func unwrap(expr string) string {
	expr = strings.TrimSpace(expr)

	if strings.HasPrefix(expr, "{{") && strings.HasSuffix(expr, "}}") {
		expr = strings.TrimSpace(expr[2 : len(expr)-2])
	}

	return expr
}

// funcMap returns the template helper functions available in expressions.
func funcMap() template.FuncMap {
	return template.FuncMap{
		"not": func(v bool) bool { return !v },
		"and": func(a, b bool) bool { return a && b },
		"or":  func(a, b bool) bool { return a || b },
		"eq":  func(a, b string) bool { return a == b },
		"ne":  func(a, b string) bool { return a != b },
	}
}