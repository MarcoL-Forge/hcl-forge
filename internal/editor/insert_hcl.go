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
	AttributePath     string
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

	if e.AttributePath != "" {
		return e.applyWithAttributePath(data, original)
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

	snippetSyntax, snippetSyntaxDiags := hclsyntax.ParseConfig([]byte(e.HCL), "snippet.tf", hcl.InitialPos)
	if snippetSyntaxDiags.HasErrors() {
		return nil, EditResult{}, fmt.Errorf("parse snippet hcl (syntax): %s", snippetSyntaxDiags.Error())
	}

	synBody, ok := snippetSyntax.Body.(*hclsyntax.Body)
	if !ok {
		return nil, EditResult{}, fmt.Errorf("unexpected snippet syntax body type %T", snippetSyntax.Body)
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

	changed := applyBodyEntriesPreserve(targetBody, snippetFile.Body(), synBody)
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

// applyBodyEntriesPreserve merges entries from the source hclwrite body into
// the target while preserving the ordering and interleaving of attributes
// and blocks as found in the source syntax body. The syntax parameter may be
// nil; when present it is used to determine original positions for ordering.
func applyBodyEntriesPreserve(target, source *hclwrite.Body, syntax *hclsyntax.Body) bool {
	changed := false

	// Build an ordered list of items (attributes and blocks) based on syntax
	// positions when available; otherwise fall back to source order.
	type item struct {
		kind     string // "attr" or "block"
		name     string // attr name
		blk      *hclwrite.Block
		synBlk   *hclsyntax.Block
		startPos int
	}

	items := make([]item, 0)

	if syntax != nil {
		// attributes
		for name, a := range syntax.Attributes {
			items = append(items, item{kind: "attr", name: name, startPos: a.NameRange.Start.Byte})
		}
		// blocks
		for _, b := range syntax.Blocks {
			items = append(items, item{kind: "block", synBlk: b, startPos: b.TypeRange.Start.Byte})
		}
		sort.Slice(items, func(i, j int) bool { return items[i].startPos < items[j].startPos })
	} else {
		// Fallback: attributes first (in source map iteration order is undefined), then blocks in source order.
		for name := range source.Attributes() {
			items = append(items, item{kind: "attr", name: name})
		}
		for _, b := range source.Blocks() {
			items = append(items, item{kind: "block", blk: b})
		}
	}

	// When matching source blocks by type/labels, handle repeated blocks by
	// tracking consumption counts so we pick the next occurrence in source.
	consumed := map[string]int{}

	for _, it := range items {
		if it.kind == "attr" {
			// Ensure attribute exists in source
			attr := source.GetAttribute(it.name)
			if attr == nil {
				continue
			}

			newTokens := attr.Expr().BuildTokens(nil)
			existing := target.GetAttribute(it.name)
			if existing != nil && tokensEqual(existing.Expr().BuildTokens(nil), newTokens) {
				continue
			}

			target.SetAttributeRaw(it.name, newTokens)
			changed = true
			continue
		}

		// Block handling
		var srcBlock *hclwrite.Block
		var synBlock *hclsyntax.Block

		if it.blk != nil {
			srcBlock = it.blk
		} else if it.synBlk != nil {
			// Find the corresponding source block by matching type/labels and
			// picking the Nth occurrence.
			typ := it.synBlk.Type
			labels := append([]string(nil), it.synBlk.Labels...)
			key := typ + "|" + strings.Join(labels, "\x00")
			idx := consumed[key]
			for _, b := range source.Blocks() {
				if b.Type() != typ {
					continue
				}
				bl := b.Labels()
				if len(bl) != len(labels) {
					continue
				}
				match := true
				for i := range bl {
					if bl[i] != labels[i] {
						match = false
						break
					}
				}
				if !match {
					continue
				}
				if idx == 0 {
					srcBlock = b
					break
				}
				idx--
			}
			consumed[key] = consumed[key] + 1
			synBlock = it.synBlk
		}

		if srcBlock == nil {
			continue
		}

		existing := findDirectMatchingBlock(target, srcBlock.Type(), srcBlock.Labels())
		if existing != nil {
			// Recurse: find corresponding syntax child body if available.
			var childSyn *hclsyntax.Body
			if synBlock != nil {
				childSyn = synBlock.Body
			}
			if applyBodyEntriesPreserve(existing.Body(), srcBlock.Body(), childSyn) {
				changed = true
			}
			continue
		}

		// Append new block preserving formatting separation.
		target.AppendNewline()
		if synBlock != nil {
			cloneBlockPreserveWithSyntax(target, srcBlock, synBlock)
		} else {
			cloneBlockPreserveWithSyntax(target, srcBlock, nil)
		}
		changed = true
	}

	return changed
}

func cloneBlockPreserveWithSyntax(target *hclwrite.Body, source *hclwrite.Block, syn *hclsyntax.Block) {
	newBlock := target.AppendNewBlock(source.Type(), source.Labels())
	var childSyn *hclsyntax.Body
	if syn != nil {
		childSyn = syn.Body
	}
	applyBodyEntriesPreserve(newBlock.Body(), source.Body(), childSyn)
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
