package editor

import (
	"fmt"
	"sort"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

type BlockSelector struct {
	Type   string
	Labels []string
}

type InsertHCLEdit struct {
	HCL         string
	TargetBlock *BlockSelector
}

func (e InsertHCLEdit) Apply(data []byte) ([]byte, EditResult, error) {
	if e.HCL == "" {
		return nil, EditResult{}, fmt.Errorf("hcl snippet cannot be empty")
	}

	file, diags := hclwrite.ParseConfig(data, "input.tf", hcl.InitialPos)
	if diags.HasErrors() {
		return nil, EditResult{}, fmt.Errorf("parse input hcl: %s", diags.Error())
	}

	snippetFile, snippetDiags := hclwrite.ParseConfig([]byte(e.HCL), "snippet.tf", hcl.InitialPos)
	if snippetDiags.HasErrors() {
		return nil, EditResult{}, fmt.Errorf("parse snippet hcl: %s", snippetDiags.Error())
	}

	targetBody := file.Body()
	if e.TargetBlock != nil {
		target := findMatchingBlock(file.Body(), *e.TargetBlock)
		if target == nil {
			return nil, EditResult{}, fmt.Errorf("target block not found: type=%q labels=%v", e.TargetBlock.Type, e.TargetBlock.Labels)
		}
		targetBody = target.Body()
	}

	changed := applyBodyEntries(targetBody, snippetFile.Body())
	if !changed {
		return data, EditResult{Changed: false, Occurrences: 0, Message: "no snippet entries found"}, nil
	}

	return file.Bytes(), EditResult{Changed: true, Occurrences: 1, Message: "insert hcl applied"}, nil
}

func findMatchingBlock(body *hclwrite.Body, selector BlockSelector) *hclwrite.Block {
	for _, block := range body.Blocks() {
		if blockMatches(block, selector) {
			return block
		}

		if nested := findMatchingBlock(block.Body(), selector); nested != nil {
			return nested
		}
	}

	return nil
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
		target.SetAttributeRaw(name, attr.Expr().BuildTokens(nil))
		changed = true
	}

	for _, block := range source.Blocks() {
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
