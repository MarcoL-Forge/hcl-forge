package editor

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

type DeleteHCLEdit struct {
	TargetBlock *BlockSelector
	Attribute   string
	DeleteAll   bool
}

func (e DeleteHCLEdit) Apply(data []byte) ([]byte, EditResult, error) {
	file, diags := hclwrite.ParseConfig(data, "input.tf", hcl.InitialPos)
	if diags.HasErrors() {
		return nil, EditResult{}, fmt.Errorf("parse input hcl: %s", diags.Error())
	}

	if e.Attribute != "" {
		targetBodies := []*hclwrite.Body{file.Body()}
		if e.TargetBlock != nil {
			if e.DeleteAll {
				targetBodies = findMatchingBodies(file.Body(), *e.TargetBlock)
			} else {
				target := findMatchingBlock(file.Body(), *e.TargetBlock)
				if target == nil {
					return nil, EditResult{}, fmt.Errorf("target block not found: type=%q labels=%v", e.TargetBlock.Type, e.TargetBlock.Labels)
				}
				targetBodies = []*hclwrite.Body{target.Body()}
			}

			if len(targetBodies) == 0 {
				return nil, EditResult{}, fmt.Errorf("target block not found: type=%q labels=%v", e.TargetBlock.Type, e.TargetBlock.Labels)
			}
		} else if e.DeleteAll {
			targetBodies = collectAllBodies(file.Body())
		}

		deleted := 0
		for _, body := range targetBodies {
			if body.GetAttribute(e.Attribute) == nil {
				continue
			}
			body.RemoveAttribute(e.Attribute)
			deleted++

			if !e.DeleteAll {
				break
			}
		}

		if deleted == 0 {
			return data, EditResult{Changed: false, Occurrences: 0, Message: "attribute not found"}, nil
		}

		return file.Bytes(), EditResult{Changed: true, Occurrences: deleted, Message: "attribute deleted"}, nil
	}

	if e.TargetBlock == nil {
		return nil, EditResult{}, fmt.Errorf("delete_hcl requires attribute or block selector")
	}

	removed := 0
	if e.DeleteAll {
		removed = removeAllMatchingBlocks(file.Body(), *e.TargetBlock)
	} else if removeFirstMatchingBlock(file.Body(), *e.TargetBlock) {
		removed = 1
	}

	if removed == 0 {
		return data, EditResult{Changed: false, Occurrences: 0, Message: "block not found"}, nil
	}

	return file.Bytes(), EditResult{Changed: true, Occurrences: removed, Message: "block deleted"}, nil
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

func removeAllMatchingBlocks(body *hclwrite.Body, selector BlockSelector) int {
	removed := 0

	for _, block := range body.Blocks() {
		removed += removeAllMatchingBlocks(block.Body(), selector)

		if blockMatches(block, selector) {
			body.RemoveBlock(block)
			removed++
		}
	}

	return removed
}

func findMatchingBodies(body *hclwrite.Body, selector BlockSelector) []*hclwrite.Body {
	bodies := make([]*hclwrite.Body, 0)

	for _, block := range body.Blocks() {
		if blockMatches(block, selector) {
			bodies = append(bodies, block.Body())
		}

		bodies = append(bodies, findMatchingBodies(block.Body(), selector)...)
	}

	return bodies
}

func collectAllBodies(body *hclwrite.Body) []*hclwrite.Body {
	bodies := []*hclwrite.Body{body}

	for _, block := range body.Blocks() {
		bodies = append(bodies, collectAllBodies(block.Body())...)
	}

	return bodies
}
