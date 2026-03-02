// Package manipulator provides a high-level API for reading and mutating
// Terraform HCL files using hashicorp/hcl/v2/hclwrite.
// All operations preserve existing formatting, whitespace, and comments.
package manipulator

import (
	"fmt"
	"os"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

// Manipulator wraps an hclwrite.File and exposes safe, high-level mutation
// operations for Terraform HCL files.
type Manipulator struct {
	path string
	file *hclwrite.File
}

// New reads a .tf file from disk and returns a Manipulator.
func New(path string) (*Manipulator, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %q: %w", path, err)
	}

	return NewFromBytes(src, path)
}

// NewFromBytes parses HCL from a byte slice.
// filename is used only in diagnostic error messages.
func NewFromBytes(src []byte, filename string) (*Manipulator, error) {
	f, diags := hclwrite.ParseConfig(src, filename, hcl.InitialPos)
	if diags.HasErrors() {
		return nil, fmt.Errorf("parsing HCL %q: %s", filename, diags.Error())
	}

	return &Manipulator{path: filename, file: f}, nil
}

// Bytes returns the current HCL content as a byte slice.
func (m *Manipulator) Bytes() []byte {
	return m.file.Bytes()
}

// WriteTo writes the current content to the given path.
func (m *Manipulator) WriteTo(path string) error {
	if err := os.WriteFile(path, m.file.Bytes(), 0o644); err != nil {
		return fmt.Errorf("writing %q: %w", path, err)
	}

	return nil
}

// Blocks returns all top-level blocks in the file.
func (m *Manipulator) Blocks() []*hclwrite.Block {
	return m.file.Body().Blocks()
}

// SetAttributeValue sets a typed cty value on an attribute in the matching block.
// Use cty.StringVal, cty.NumberIntVal, cty.True/False, cty.ListVal, etc.
func (m *Manipulator) SetAttributeValue(blockType string, labels []string, attr string, val cty.Value) error {
	block, err := m.requireBlock(blockType, labels)
	if err != nil {
		return fmt.Errorf("SetAttributeValue: %w", err)
	}

	block.Body().SetAttributeValue(attr, val)

	return nil
}

// SetAttributeRaw sets an attribute to a raw HCL expression string.
// Use this for HCL references such as var.x, local.y, module.z.output.
func (m *Manipulator) SetAttributeRaw(blockType string, labels []string, attr, rawExpr string) error {
	block, err := m.requireBlock(blockType, labels)
	if err != nil {
		return fmt.Errorf("SetAttributeRaw: %w", err)
	}

	tokens := hclwrite.Tokens{
		{Type: 9 /*TokenIdent*/, Bytes: []byte(rawExpr)},
		{Type: 14 /*TokenNewline*/, Bytes: []byte("\n")},
	}
	block.Body().SetAttributeRaw(attr, tokens)

	return nil
}

// RemoveAttribute removes a named attribute from the matching block.
// No error is returned if the attribute does not exist.
func (m *Manipulator) RemoveAttribute(blockType string, labels []string, attr string) error {
	block, err := m.requireBlock(blockType, labels)
	if err != nil {
		return fmt.Errorf("RemoveAttribute: %w", err)
	}

	block.Body().RemoveAttribute(attr)

	return nil
}

// RemoveBlock removes a top-level block identified by type and labels.
func (m *Manipulator) RemoveBlock(blockType string, labels []string) error {
	block, err := m.requireBlock(blockType, labels)
	if err != nil {
		return fmt.Errorf("RemoveBlock: %w", err)
	}

	m.file.Body().RemoveBlock(block)

	return nil
}

// RenameBlock changes the last label of a matching block.
// Because hclwrite does not support in-place label mutation, the block is
// fully rebuilt with the new labels and appended in place of the original.
func (m *Manipulator) RenameBlock(blockType string, labels []string, newName string) error {
	block, err := m.requireBlock(blockType, labels)
	if err != nil {
		return fmt.Errorf("RenameBlock: %w", err)
	}

	newLabels := make([]string, len(labels))
	copy(newLabels, labels)
	newLabels[len(newLabels)-1] = newName

	newBlock := hclwrite.NewBlock(blockType, newLabels)
	copyBody(newBlock.Body(), block.Body())

	m.file.Body().RemoveBlock(block)
	m.file.Body().AppendBlock(newBlock)

	return nil
}

// AppendBlock appends a new empty block and returns it for further mutation.
func (m *Manipulator) AppendBlock(blockType string, labels ...string) *hclwrite.Block {
	return m.file.Body().AppendNewBlock(blockType, labels)
}

// AppendRawBlock parses raw HCL and appends all blocks it contains.
func (m *Manipulator) AppendRawBlock(src []byte) error {
	tmp, diags := hclwrite.ParseConfig(src, "<snippet>", hcl.InitialPos)
	if diags.HasErrors() {
		return fmt.Errorf("parsing raw block: %s", diags.Error())
	}

	for _, b := range tmp.Body().Blocks() {
		newBlock := hclwrite.NewBlock(b.Type(), b.Labels())
		copyBody(newBlock.Body(), b.Body())
		m.file.Body().AppendBlock(newBlock)
	}

	return nil
}

// CopyBlockFrom copies a block identified by blockType and labels from another
// Manipulator into this one. An optional newName renames the last label.
func (m *Manipulator) CopyBlockFrom(src *Manipulator, blockType string, labels []string, newName string) error {
	block, err := src.requireBlock(blockType, labels)
	if err != nil {
		return fmt.Errorf("CopyBlockFrom: source %w", err)
	}

	destLabels := make([]string, len(labels))
	copy(destLabels, labels)

	if newName != "" {
		destLabels[len(destLabels)-1] = newName
	}

	newBlock := hclwrite.NewBlock(blockType, destLabels)
	copyBody(newBlock.Body(), block.Body())
	m.file.Body().AppendBlock(newBlock)

	return nil
}

// SetNestedAttributeValue sets an attribute inside a nested block, creating
// the nested block if it does not already exist.
func (m *Manipulator) SetNestedAttributeValue(blockType string, labels []string, nestedType, attr string, val cty.Value) error {
	block, err := m.requireBlock(blockType, labels)
	if err != nil {
		return fmt.Errorf("SetNestedAttributeValue: %w", err)
	}

	nested := findNestedBlock(block.Body(), nestedType)
	if nested == nil {
		nested = block.Body().AppendNewBlock(nestedType, nil)
	}

	nested.Body().SetAttributeValue(attr, val)

	return nil
}

// RemoveNestedBlock removes a nested block by type from a parent block.
func (m *Manipulator) RemoveNestedBlock(blockType string, labels []string, nestedType string) error {
	block, err := m.requireBlock(blockType, labels)
	if err != nil {
		return fmt.Errorf("RemoveNestedBlock: %w", err)
	}

	nested := findNestedBlock(block.Body(), nestedType)
	if nested == nil {
		return fmt.Errorf("RemoveNestedBlock: nested block %q not found in %s %v", nestedType, blockType, labels)
	}

	block.Body().RemoveBlock(nested)

	return nil
}

// GetAttributeRaw returns the raw token bytes of an attribute expression.
// Useful for inspecting values before or after transformation.
func (m *Manipulator) GetAttributeRaw(blockType string, labels []string, attr string) ([]byte, error) {
	block, err := m.requireBlock(blockType, labels)
	if err != nil {
		return nil, fmt.Errorf("GetAttributeRaw: %w", err)
	}

	a := block.Body().GetAttribute(attr)
	if a == nil {
		return nil, fmt.Errorf("GetAttributeRaw: attribute %q not found in %s %v", attr, blockType, labels)
	}

	return a.Expr().BuildTokens(nil).Bytes(), nil
}

// --- internal helpers ---

// requireBlock looks up a block and returns an error if it is not found.
func (m *Manipulator) requireBlock(blockType string, labels []string) (*hclwrite.Block, error) {
	block := m.findBlock(blockType, labels)
	if block == nil {
		return nil, fmt.Errorf("block %s %v not found", blockType, labels)
	}

	return block, nil
}

// findBlock returns the first top-level block matching blockType and labels, or nil.
func (m *Manipulator) findBlock(blockType string, labels []string) *hclwrite.Block {
	for _, block := range m.file.Body().Blocks() {
		if block.Type() != blockType {
			continue
		}

		if labelsMatch(block.Labels(), labels) {
			return block
		}
	}

	return nil
}

// labelsMatch returns true when a and b have identical length and content.
func labelsMatch(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}

	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}

	return true
}

// findNestedBlock returns the first nested block with the given type, or nil.
func findNestedBlock(body *hclwrite.Body, blockType string) *hclwrite.Block {
	for _, b := range body.Blocks() {
		if b.Type() == blockType {
			return b
		}
	}

	return nil
}

// copyBody deep-copies all attributes and nested blocks from src into dst.
func copyBody(dst, src *hclwrite.Body) {
	for name, attr := range src.Attributes() {
		dst.SetAttributeRaw(name, attr.Expr().BuildTokens(nil))
	}

	for _, block := range src.Blocks() {
		nb := dst.AppendNewBlock(block.Type(), block.Labels())
		copyBody(nb.Body(), block.Body())
	}
}