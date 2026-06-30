package editor

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

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
					return data, EditResult{Changed: false, Occurrences: 0, Message: "target block not found"}, nil
				}
				targetBodies = []*hclwrite.Body{target.Body()}
			}

			if len(targetBodies) == 0 {
				return data, EditResult{Changed: false, Occurrences: 0, Message: "target block not found"}, nil
			}
		} else if e.DeleteAll {
			targetBodies = collectAllBodies(file.Body())
		}

		deleted := 0
		usePattern := hasGlobPattern(e.Attribute)
		for _, body := range targetBodies {
			if usePattern {
				names := sortedAttributeNames(body)
				for _, name := range names {
					if !matchesPattern(e.Attribute, name) {
						continue
					}

					body.RemoveAttribute(name)
					deleted++

					if !e.DeleteAll {
						break
					}
				}

				if deleted > 0 && !e.DeleteAll {
					break
				}

				continue
			}

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
	return removeFirstMatchingBlockWithParents(body, selector, nil)
}

func removeFirstMatchingBlockWithParents(body *hclwrite.Body, selector BlockSelector, ancestry []ParentSelector) bool {
	for _, block := range body.Blocks() {
		if blockMatchesDeleteSelector(block, selector) && parentsMatchDeleteSelector(ancestry, selector.Parents) {
			body.RemoveBlock(block)
			return true
		}

		nextAncestry := append(append([]ParentSelector(nil), ancestry...), ParentSelector{
			Type:   block.Type(),
			Labels: append([]string(nil), block.Labels()...),
		})

		if removed := removeFirstMatchingBlockWithParents(block.Body(), selector, nextAncestry); removed {
			return true
		}
	}

	return false
}

func removeAllMatchingBlocks(body *hclwrite.Body, selector BlockSelector) int {
	return removeAllMatchingBlocksWithParents(body, selector, nil)
}

func removeAllMatchingBlocksWithParents(body *hclwrite.Body, selector BlockSelector, ancestry []ParentSelector) int {
	removed := 0

	for _, block := range body.Blocks() {
		nextAncestry := append(append([]ParentSelector(nil), ancestry...), ParentSelector{
			Type:   block.Type(),
			Labels: append([]string(nil), block.Labels()...),
		})

		removed += removeAllMatchingBlocksWithParents(block.Body(), selector, nextAncestry)

		if blockMatchesDeleteSelector(block, selector) && parentsMatchDeleteSelector(ancestry, selector.Parents) {
			body.RemoveBlock(block)
			removed++
		}
	}

	return removed
}

func findMatchingBodies(body *hclwrite.Body, selector BlockSelector) []*hclwrite.Body {
	return findMatchingBodiesWithParents(body, selector, nil)
}

func findMatchingBodiesWithParents(body *hclwrite.Body, selector BlockSelector, ancestry []ParentSelector) []*hclwrite.Body {
	bodies := make([]*hclwrite.Body, 0)

	for _, block := range body.Blocks() {
		if blockMatchesDeleteSelector(block, selector) && parentsMatchDeleteSelector(ancestry, selector.Parents) {
			bodies = append(bodies, block.Body())
		}

		nextAncestry := append(append([]ParentSelector(nil), ancestry...), ParentSelector{
			Type:   block.Type(),
			Labels: append([]string(nil), block.Labels()...),
		})

		bodies = append(bodies, findMatchingBodiesWithParents(block.Body(), selector, nextAncestry)...)
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

func sortedAttributeNames(body *hclwrite.Body) []string {
	attrs := body.Attributes()
	names := make([]string, 0, len(attrs))
	for name := range attrs {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func hasGlobPattern(pattern string) bool {
	return strings.ContainsAny(pattern, "*?[")
}

func matchesPattern(pattern, value string) bool {
	if !hasGlobPattern(pattern) {
		return pattern == value
	}

	matched, err := filepath.Match(pattern, value)
	return err == nil && matched
}

func blockMatchesDeleteSelector(block *hclwrite.Block, selector BlockSelector) bool {
	if !matchesPattern(selector.Type, block.Type()) {
		return false
	}

	labels := block.Labels()
	if len(selector.Labels) != len(labels) {
		return false
	}

	for i := range labels {
		if !matchesPattern(selector.Labels[i], labels[i]) {
			return false
		}
	}

	return true
}

func parentsMatchDeleteSelector(ancestry, expected []ParentSelector) bool {
	if len(expected) == 0 {
		return true
	}

	if len(ancestry) != len(expected) {
		return false
	}

	for i := range ancestry {
		if !matchesPattern(expected[i].Type, ancestry[i].Type) {
			return false
		}

		if len(ancestry[i].Labels) != len(expected[i].Labels) {
			return false
		}

		for j := range ancestry[i].Labels {
			if !matchesPattern(expected[i].Labels[j], ancestry[i].Labels[j]) {
				return false
			}
		}
	}

	return true
}
