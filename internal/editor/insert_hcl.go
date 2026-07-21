package editor

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/MarcoL-Forge/hcl-forge/internal/logging"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

type BlockSelector struct {
	Type    string
	Labels  []string
	Parents []ParentSelector
}

type ParentSelector struct {
	Type   string
	Labels []string
}

type InsertGuard struct {
	IfTargetExists  bool
	IfTargetMissing bool
}

type InsertPlacement struct {
	Mode      string
	Attribute string
	Strict    bool
}

type InsertHCLEdit struct {
	HCL               string
	TargetBlock       *BlockSelector
	EnsureTargetBlock bool
	Guard             *InsertGuard
	Placement         *InsertPlacement
}

func (e InsertHCLEdit) Apply(data []byte) ([]byte, EditResult, error) {
	return e.ApplyWithOriginal(data, data)
}

func (e InsertHCLEdit) ApplyWithOriginal(data []byte, original []byte) ([]byte, EditResult, error) {
	logger := logging.Default()
	logger.Debug("insert_hcl_start", map[string]any{
		"has_target":    e.TargetBlock != nil,
		"ensure_target": e.EnsureTargetBlock,
	})

	if e.HCL == "" {
		return nil, EditResult{}, fmt.Errorf("hcl snippet cannot be empty")
	}

	file, diags := hclwrite.ParseConfig(data, "input.tf", hcl.InitialPos)
	if diags.HasErrors() {
		return nil, EditResult{}, fmt.Errorf("parse input hcl: %s", diags.Error())
	}

	originalFile, originalDiags := hclwrite.ParseConfig(original, "original.tf", hcl.InitialPos)
	if originalDiags.HasErrors() {
		return nil, EditResult{}, fmt.Errorf("parse original hcl: %s", originalDiags.Error())
	}

	snippetFile, snippetDiags := hclwrite.ParseConfig([]byte(e.HCL), "snippet.tf", hcl.InitialPos)
	if snippetDiags.HasErrors() {
		return nil, EditResult{}, fmt.Errorf("parse snippet hcl: %s", snippetDiags.Error())
	}

	targetBody := file.Body()
	createdTarget := false
	var targetPosition bodyPosition
	hasTargetPosition := false
	if e.TargetBlock != nil {
		logger.Debug("insert_hcl_resolve_target", map[string]any{
			"type":   e.TargetBlock.Type,
			"labels": e.TargetBlock.Labels,
		})
		if position, ok := findFirstMatchingBodyPositionExact(originalFile.Body(), *e.TargetBlock); ok {
			targetPosition = position
			hasTargetPosition = true
		}

		originalTarget := findMatchingBlock(originalFile.Body(), *e.TargetBlock)
		target := targetBlockFromOriginal(file.Body(), originalFile.Body(), *e.TargetBlock)
		if target == nil {
			logger.Debug("insert_hcl_target_missing", map[string]any{"type": e.TargetBlock.Type, "labels": e.TargetBlock.Labels})
			if e.Guard != nil && e.Guard.IfTargetExists {
				return data, EditResult{Changed: false, Occurrences: 0, Message: "guard skipped: target block missing"}, nil
			}

			if e.EnsureTargetBlock {
				target = ensureBlockPath(file.Body(), *e.TargetBlock)
				createdTarget = true
				logger.Debug("insert_hcl_target_created", map[string]any{"type": e.TargetBlock.Type, "labels": e.TargetBlock.Labels})
			} else if e.Guard != nil && e.Guard.IfTargetMissing {
				return data, EditResult{Changed: false, Occurrences: 0, Message: "guard matched: target missing and ensure_target_block=false"}, nil
			} else {
				return nil, EditResult{}, fmt.Errorf("target block not found: type=%q labels=%v", e.TargetBlock.Type, e.TargetBlock.Labels)
			}
		} else if e.Guard != nil && e.Guard.IfTargetMissing {
			if originalTarget != nil {
				return data, EditResult{Changed: false, Occurrences: 0, Message: "guard skipped: target block exists"}, nil
			}
		} else if e.Guard != nil && e.Guard.IfTargetExists && originalTarget == nil {
			return data, EditResult{Changed: false, Occurrences: 0, Message: "guard skipped: target block missing"}, nil
		}

		if e.Guard != nil && e.Guard.IfTargetMissing && originalTarget != nil {
			return data, EditResult{Changed: false, Occurrences: 0, Message: "guard skipped: target block exists"}, nil
		}
		targetBody = target.Body()
	}

	placement := InsertPlacement{Mode: "append"}
	hasExplicitPlacement := e.Placement != nil
	if hasExplicitPlacement {
		placement = *e.Placement
		if placement.Mode == "" {
			placement.Mode = "append"
		}
	}

	if e.TargetBlock != nil && hasTargetPosition && hasExplicitPlacement {
		inserted, changed, err := insertAtPlacement(data, targetPosition, e.HCL, placement)
		if err != nil {
			return nil, EditResult{}, err
		}

		if changed {
			logger.Debug("insert_hcl_completed", map[string]any{"changed": true})
			return inserted, EditResult{Changed: true, Occurrences: 1, Message: "insert hcl applied"}, nil
		}
	}

	changed := applyBodyEntries(targetBody, snippetFile.Body())
	if !changed && !createdTarget {
		return data, EditResult{Changed: false, Occurrences: 0, Message: "no snippet entries found"}, nil
	}

	logger.Debug("insert_hcl_completed", map[string]any{"changed": changed || createdTarget})

	return file.Bytes(), EditResult{Changed: true, Occurrences: 1, Message: "insert hcl applied"}, nil
}

func targetBlockFromOriginal(currentRoot, originalRoot *hclwrite.Body, selector BlockSelector) *hclwrite.Block {
	position, ok := findFirstMatchingBodyPositionExact(originalRoot, selector)
	if !ok {
		return nil
	}

	return blockAtPosition(currentRoot, position)
}

func findFirstMatchingBodyPositionExact(root *hclwrite.Body, selector BlockSelector) (bodyPosition, bool) {
	return findFirstMatchingBodyPositionExactWithParents(root, selector, nil, nil)
}

func findFirstMatchingBodyPositionExactWithParents(
	body *hclwrite.Body,
	selector BlockSelector,
	ancestry []ParentSelector,
	path bodyPosition,
) (bodyPosition, bool) {
	for index, block := range body.Blocks() {
		nextPath := append(append(bodyPosition(nil), path...), index)
		if blockMatches(block, selector) && parentsMatch(ancestry, selector.Parents) {
			return nextPath, true
		}

		nextAncestry := append(append([]ParentSelector(nil), ancestry...), ParentSelector{
			Type:   block.Type(),
			Labels: append([]string(nil), block.Labels()...),
		})

		if nested, ok := findFirstMatchingBodyPositionExactWithParents(block.Body(), selector, nextAncestry, nextPath); ok {
			return nested, true
		}
	}

	return nil, false
}

func blockAtPosition(root *hclwrite.Body, position bodyPosition) *hclwrite.Block {
	if len(position) == 0 {
		return nil
	}

	current := root
	for i, idx := range position {
		blocks := current.Blocks()
		if idx < 0 || idx >= len(blocks) {
			return nil
		}

		block := blocks[idx]
		if i == len(position)-1 {
			return block
		}

		current = block.Body()
	}

	return nil
}

func ensureBlockPath(root *hclwrite.Body, selector BlockSelector) *hclwrite.Block {
	current := root

	for _, parent := range selector.Parents {
		block := findDirectMatchingBlock(current, parent.Type, parent.Labels)
		if block == nil {
			current.AppendNewline()
			block = current.AppendNewBlock(parent.Type, parent.Labels)
		}
		current = block.Body()
	}

	target := findDirectMatchingBlock(current, selector.Type, selector.Labels)
	if target == nil {
		current.AppendNewline()
		target = current.AppendNewBlock(selector.Type, selector.Labels)
	}

	return target
}

func findDirectMatchingBlock(body *hclwrite.Body, blockType string, labels []string) *hclwrite.Block {
	for _, block := range body.Blocks() {
		if block.Type() != blockType {
			continue
		}

		blockLabels := block.Labels()
		if len(blockLabels) != len(labels) {
			continue
		}

		matched := true
		for i := range blockLabels {
			if blockLabels[i] != labels[i] {
				matched = false
				break
			}
		}

		if matched {
			return block
		}
	}

	return nil
}

func findMatchingBlock(body *hclwrite.Body, selector BlockSelector) *hclwrite.Block {
	return findMatchingBlockWithParents(body, selector, nil)
}

func findMatchingBlockWithParents(body *hclwrite.Body, selector BlockSelector, ancestry []ParentSelector) *hclwrite.Block {
	for _, block := range body.Blocks() {
		if blockMatches(block, selector) && parentsMatch(ancestry, selector.Parents) {
			return block
		}

		nextAncestry := append(append([]ParentSelector(nil), ancestry...), ParentSelector{
			Type:   block.Type(),
			Labels: append([]string(nil), block.Labels()...),
		})

		if nested := findMatchingBlockWithParents(block.Body(), selector, nextAncestry); nested != nil {
			return nested
		}
	}

	return nil
}

func parentsMatch(ancestry, expected []ParentSelector) bool {
	if len(expected) == 0 {
		return true
	}

	if len(ancestry) != len(expected) {
		return false
	}

	for i := range ancestry {
		if ancestry[i].Type != expected[i].Type {
			return false
		}

		if len(ancestry[i].Labels) != len(expected[i].Labels) {
			return false
		}

		for j := range ancestry[i].Labels {
			if ancestry[i].Labels[j] != expected[i].Labels[j] {
				return false
			}
		}
	}

	return true
}

func blockMatches(block *hclwrite.Block, selector BlockSelector) bool {
	if block.Type() != selector.Type {
		return false
	}

	labels := block.Labels()
	if len(selector.Labels) != len(labels) {
		return false
	}

	for i := range labels {
		if labels[i] != selector.Labels[i] {
			return false
		}
	}

	return true
}

func applyBodyEntries(target, source *hclwrite.Body) bool {
	changed := false

	attributes := source.Attributes()
	attributeNames := make([]string, 0, len(attributes))
	for name := range attributes {
		attributeNames = append(attributeNames, name)
	}
	sort.Strings(attributeNames)

	for _, name := range attributeNames {
		attr := attributes[name]
		newTokens := attr.Expr().BuildTokens(nil)
		existing := target.GetAttribute(name)
		if existing != nil && tokensEqual(existing.Expr().BuildTokens(nil), newTokens) {
			continue
		}

		target.SetAttributeRaw(name, newTokens)
		changed = true
	}

	for _, block := range source.Blocks() {
		existing := findDirectMatchingBlock(target, block.Type(), block.Labels())
		if existing != nil {
			if applyBodyEntries(existing.Body(), block.Body()) {
				changed = true
			}
			continue
		}

		// Ensure appended blocks are separated from existing content.
		target.AppendNewline()
		cloneBlock(target, block)
		changed = true
	}

	return changed
}

func cloneBlock(target *hclwrite.Body, source *hclwrite.Block) {
	newBlock := target.AppendNewBlock(source.Type(), source.Labels())
	applyBodyEntries(newBlock.Body(), source.Body())
}

func tokensEqual(a, b hclwrite.Tokens) bool {
	return bytes.Equal(a.Bytes(), b.Bytes())
}

func insertAtPlacement(data []byte, targetPosition bodyPosition, snippet string, placement InsertPlacement) ([]byte, bool, error) {
	if strings.TrimSpace(snippet) == "" {
		return nil, false, fmt.Errorf("hcl snippet cannot be empty")
	}

	parsed, diags := hclsyntax.ParseConfig(data, "input.tf", hcl.InitialPos)
	if diags.HasErrors() {
		return nil, false, fmt.Errorf("parse input hcl for placement: %s", diags.Error())
	}

	rootBody, ok := parsed.Body.(*hclsyntax.Body)
	if !ok {
		return nil, false, fmt.Errorf("unexpected body type %T", parsed.Body)
	}

	target := syntaxBlockAtPosition(rootBody, targetPosition)
	if target == nil {
		return nil, false, fmt.Errorf("target block not found for placement")
	}

	indent := bodyIndent(target)
	formattedSnippet := formatSnippetWithIndent(snippet, indent)
	mode := placement.Mode
	if mode == "" {
		mode = "append"
	}

	startOffset := target.OpenBraceRange.End.Byte
	endOffset := target.CloseBraceRange.Start.Byte
	insertOffset := 0

	switch mode {
	case "append":
		insertOffset = lineStartOffset(data, endOffset)
	case "prepend":
		insertOffset = lineEndOffset(data, startOffset)
	case "after_attribute", "before_attribute":
		attr := target.Body.Attributes[placement.Attribute]
		if attr == nil {
			if placement.Strict {
				return nil, false, fmt.Errorf("placement attribute %q not found", placement.Attribute)
			}

			insertOffset = lineStartOffset(data, endOffset)
		} else if mode == "after_attribute" {
			insertOffset = lineEndOffset(data, attr.Range().End.Byte)
		} else {
			insertOffset = lineStartOffset(data, attr.Range().Start.Byte)
		}
	default:
		return nil, false, fmt.Errorf("unsupported insert placement mode %q", mode)
	}

	if insertOffset < startOffset || insertOffset > endOffset {
		return nil, false, fmt.Errorf("computed placement offset %d is outside target body", insertOffset)
	}

	insertText := formattedSnippet + "\n"
	out := make([]byte, 0, len(data)+len(insertText))
	out = append(out, data[:insertOffset]...)
	out = append(out, []byte(insertText)...)
	out = append(out, data[insertOffset:]...)

	return out, true, nil
}

func syntaxBlockAtPosition(root *hclsyntax.Body, position bodyPosition) *hclsyntax.Block {
	if len(position) == 0 {
		return nil
	}

	current := root
	for i, idx := range position {
		if idx < 0 || idx >= len(current.Blocks) {
			return nil
		}

		block := current.Blocks[idx]
		if i == len(position)-1 {
			return block
		}

		current = block.Body
	}

	return nil
}

func lineStartOffset(data []byte, offset int) int {
	if offset < 0 {
		return 0
	}
	if offset > len(data) {
		offset = len(data)
	}

	for offset > 0 && data[offset-1] != '\n' {
		offset--
	}

	return offset
}

func lineEndOffset(data []byte, offset int) int {
	if offset < 0 {
		return 0
	}
	if offset > len(data) {
		offset = len(data)
	}

	for offset < len(data) && data[offset] != '\n' {
		offset++
	}

	if offset < len(data) {
		offset++
	}

	return offset
}

func bodyIndent(block *hclsyntax.Block) string {
	if block.Body == nil {
		return "  "
	}

	minColumn := 0
	for _, attr := range block.Body.Attributes {
		col := attr.NameRange.Start.Column
		if col <= 1 {
			continue
		}
		if minColumn == 0 || col < minColumn {
			minColumn = col
		}
	}

	for _, child := range block.Body.Blocks {
		col := child.TypeRange.Start.Column
		if col <= 1 {
			continue
		}
		if minColumn == 0 || col < minColumn {
			minColumn = col
		}
	}

	if minColumn <= 1 {
		return "  "
	}

	return strings.Repeat(" ", minColumn-1)
}

func formatSnippetWithIndent(snippet, indent string) string {
	trimmed := strings.TrimSpace(snippet)
	if trimmed == "" {
		return ""
	}

	lines := strings.Split(trimmed, "\n")
	for i, line := range lines {
		line = strings.TrimRight(line, " \t")
		if line == "" {
			lines[i] = ""
			continue
		}

		lines[i] = indent + line
	}

	return strings.Join(lines, "\n")
}
