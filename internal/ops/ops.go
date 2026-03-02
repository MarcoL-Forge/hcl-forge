// Package ops implements each hclforge change type as a discrete, testable
// operation against a manipulator.Manipulator.
package ops

import (
	"fmt"
	"os"
	"strings"

	"github.com/Marc0l95/hclforge/internal/config"
	"github.com/Marc0l95/hclforge/internal/manipulator"
	tmpl "github.com/Marc0l95/hclforge/internal/template"
	"github.com/zclconf/go-cty/cty"
)

// Apply executes a single Change against dst, using eval for template
// rendering and sourceDir to resolve from_file paths.
// Returns nil if the change's condition evaluates to false (skip).
func Apply(dst *manipulator.Manipulator, c config.Change, eval *tmpl.Evaluator, sourceDir string) error {
	ok, err := eval.EvalCondition(c.If)
	if err != nil {
		return fmt.Errorf("evaluating condition for %s: %w", c.Type, err)
	}

	if !ok {
		return nil
	}

	switch c.Type {
	case "set_attr":
		return applySetAttr(dst, c, eval)
	case "remove_attr":
		return applyRemoveAttr(dst, c)
	case "remove_block":
		return applyRemoveBlock(dst, c)
	case "copy_block":
		return applyCopyBlock(dst, c, eval, sourceDir)
	case "add_block":
		return applyAddBlock(dst, c, sourceDir)
	default:
		return fmt.Errorf("unknown change type %q", c.Type)
	}
}

// applySetAttr sets a single attribute value, rendering any template expressions
// in the value field before applying.
func applySetAttr(dst *manipulator.Manipulator, c config.Change, eval *tmpl.Evaluator) error {
	ref, err := config.ParseBlockRef(c.Block)
	if err != nil {
		return fmt.Errorf("set_attr: %w", err)
	}

	rendered, err := eval.Render(c.Value)
	if err != nil {
		return fmt.Errorf("set_attr: rendering value %q: %w", c.Value, err)
	}

	// If the rendered value looks like an HCL reference (no surrounding quotes)
	// treat it as a raw token expression; otherwise wrap as a string value.
	if isRawExpression(rendered) {
		return dst.SetAttributeRaw(ref.Type, ref.Labels, c.Attr, rendered)
	}

	return dst.SetAttributeValue(ref.Type, ref.Labels, c.Attr, cty.StringVal(rendered))
}

// applyRemoveAttr removes an attribute from the target block.
func applyRemoveAttr(dst *manipulator.Manipulator, c config.Change) error {
	ref, err := config.ParseBlockRef(c.Block)
	if err != nil {
		return fmt.Errorf("remove_attr: %w", err)
	}

	return dst.RemoveAttribute(ref.Type, ref.Labels, c.Attr)
}

// applyRemoveBlock removes an entire block from the file.
func applyRemoveBlock(dst *manipulator.Manipulator, c config.Change) error {
	ref, err := config.ParseBlockRef(c.Block)
	if err != nil {
		return fmt.Errorf("remove_block: %w", err)
	}

	return dst.RemoveBlock(ref.Type, ref.Labels)
}

// applyCopyBlock copies a block from source_dir or from_file into dst,
// then applies any nested `with` changes to the copy.
func applyCopyBlock(dst *manipulator.Manipulator, c config.Change, eval *tmpl.Evaluator, sourceDir string) error {
	ref, err := config.ParseBlockRef(c.Block)
	if err != nil {
		return fmt.Errorf("copy_block: %w", err)
	}

	// Determine the source: from_file overrides source_dir.
	srcPath := c.FromFile
	if srcPath == "" {
		return fmt.Errorf("copy_block: from_file is required")
	}

	if !isAbsolute(srcPath) {
		srcPath = sourceDir + "/" + srcPath
	}

	src, err := manipulator.New(srcPath)
	if err != nil {
		return fmt.Errorf("copy_block: loading source file %q: %w", srcPath, err)
	}

	if err := dst.CopyBlockFrom(src, ref.Type, ref.Labels, c.Rename); err != nil {
		return fmt.Errorf("copy_block: %w", err)
	}

	// Apply nested with-changes to the newly copied block.
	if len(c.With) == 0 {
		return nil
	}

	// Resolve the target labels of the copied block.
	targetLabels := make([]string, len(ref.Labels))
	copy(targetLabels, ref.Labels)

	if c.Rename != "" {
		targetLabels[len(targetLabels)-1] = c.Rename
	}

	for _, nested := range c.With {
		nested.Block = ref.Type + "." + strings.Join(targetLabels, ".")
		if err := Apply(dst, nested, eval, sourceDir); err != nil {
			return fmt.Errorf("copy_block with[%s]: %w", nested.Type, err)
		}
	}

	return nil
}

// applyAddBlock reads a .tf snippet file and appends its blocks to dst.
func applyAddBlock(dst *manipulator.Manipulator, c config.Change, sourceDir string) error {
	path := c.FromFile
	if !isAbsolute(path) {
		path = sourceDir + "/" + path
	}

	src, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("add_block: reading snippet %q: %w", path, err)
	}

	return dst.AppendRawBlock(src)
}

// isRawExpression returns true when a value should be treated as an HCL
// reference expression rather than a plain string literal.
// Heuristic: starts with "var.", "local.", "module.", or "data.".
func isRawExpression(val string) bool {
	prefixes := []string{"var.", "local.", "module.", "data.", "each.", "self."}
	for _, p := range prefixes {
		if strings.HasPrefix(val, p) {
			return true
		}
	}

	return false
}

// isAbsolute returns true if path starts with "/" or "~".
func isAbsolute(path string) bool {
	return strings.HasPrefix(path, "/") || strings.HasPrefix(path, "~")
}