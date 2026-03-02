// Package engine orchestrates the hclforge transformation pipeline.
// It reads the spec, copies source files to the target directory, applies
// each change in order, and writes the results. Dry-run mode prints a diff
// without writing anything to disk.
package engine

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Marc0l95/hclforge/internal/config"
	"github.com/Marc0l95/hclforge/internal/manipulator"
	"github.com/Marc0l95/hclforge/internal/ops"
	tmpl "github.com/Marc0l95/hclforge/internal/template"
)

// Engine applies a Spec to produce output Terraform files.
type Engine struct {
	spec *config.Spec
}

// New creates an Engine for the given spec.
func New(spec *config.Spec) *Engine {
	return &Engine{spec: spec}
}

// Run executes the transformation pipeline.
// When dryRun is true, changes are described to stdout but no files are written.
func (e *Engine) Run(dryRun bool) error {
	if err := e.spec.Validate(); err != nil {
		return fmt.Errorf("invalid spec: %w", err)
	}

	if !dryRun {
		if err := os.MkdirAll(e.spec.TargetDir, 0o755); err != nil {
			return fmt.Errorf("creating target dir %q: %w", e.spec.TargetDir, err)
		}
	}

	eval := tmpl.New(e.spec.Vars, e.spec.Flags)

	for _, f := range e.spec.Files {
		if err := e.processFile(f, eval, dryRun); err != nil {
			return fmt.Errorf("processing %q: %w", f.Path, err)
		}
	}

	return nil
}

// processFile handles one FileSpec: load source, apply changes, write target.
func (e *Engine) processFile(f config.FileSpec, eval *tmpl.Evaluator, dryRun bool) error {
	srcPath := filepath.Join(e.spec.SourceDir, f.Path)
	dstPath := filepath.Join(e.spec.TargetDir, f.Path)

	m, err := manipulator.New(srcPath)
	if err != nil {
		return fmt.Errorf("loading source file: %w", err)
	}

	for i, c := range f.Changes {
		if err := ops.Apply(m, c, eval, e.spec.SourceDir); err != nil {
			return fmt.Errorf("change[%d] (%s): %w", i, c.Type, err)
		}
	}

	if dryRun {
		fmt.Printf("\n--- dry-run: %s → %s ---\n", srcPath, dstPath)
		fmt.Println(string(m.Bytes()))

		return nil
	}

	if err := os.MkdirAll(filepath.Dir(dstPath), 0o755); err != nil {
		return fmt.Errorf("creating output dir for %q: %w", dstPath, err)
	}

	if err := m.WriteTo(dstPath); err != nil {
		return fmt.Errorf("writing output: %w", err)
	}

	fmt.Printf("✓ %s\n", dstPath)

	return nil
}