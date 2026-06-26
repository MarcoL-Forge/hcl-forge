package editor

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

type DeleteHCLEdit struct {
	TargetBlock *BlockSelector
	Attribute   string
}

func (e DeleteHCLEdit) Apply(data []byte) ([]byte, EditResult, error) {
	file, diags := hclwrite.ParseConfig(data, "input.tf", hcl.InitialPos)
	if diags.HasErrors() {
		return nil, EditResult{}, fmt.Errorf("parse input hcl: %s", diags.Error())
	}

	if e.Attribute != "" {
		targetBody := file.Body()
		if e.TargetBlock != nil {
			target := findMatchingBlock(file.Body(), *e.TargetBlock)
			if target == nil {
				return nil, EditResult{}, fmt.Errorf("target block not found: type=%q labels=%v", e.TargetBlock.Type, e.TargetBlock.Labels)
			}
			targetBody = target.Body()
		}

		if targetBody.GetAttribute(e.Attribute) == nil {
			return data, EditResult{Changed: false, Occurrences: 0, Message: "attribute not found"}, nil
		}

		targetBody.RemoveAttribute(e.Attribute)
		return file.Bytes(), EditResult{Changed: true, Occurrences: 1, Message: "attribute deleted"}, nil
	}

	if e.TargetBlock == nil {
		return nil, EditResult{}, fmt.Errorf("delete_hcl requires attribute or block selector")
	}

	if !removeFirstMatchingBlock(file.Body(), *e.TargetBlock) {
		return data, EditResult{Changed: false, Occurrences: 0, Message: "block not found"}, nil
	}

	return file.Bytes(), EditResult{Changed: true, Occurrences: 1, Message: "block deleted"}, nil
}

func removeFirstMatchingBlock(body *hclwrite.Body, selector BlockSelector) bool {
	for _, block := range body.Blocks() {
		if blockMatches(block, selector) {
			body.RemoveBlock(block)
			return true
		}

		if removed := removeFirstMatchingBlock(block.Body(), selector); removed {
			return true
		}
	}

	return false
}
