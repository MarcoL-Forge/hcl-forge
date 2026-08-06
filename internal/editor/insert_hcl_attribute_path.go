package editor

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

func (e InsertHCLEdit) applyWithAttributePath(data []byte, original []byte) ([]byte, EditResult, error) {
	path, err := parseAttributePath(e.AttributePath)
	if err != nil {
		return nil, EditResult{}, err
	}

	file, diags := hclwrite.ParseConfig(data, "input.tf", hcl.InitialPos)
	if diags.HasErrors() {
		return nil, EditResult{}, fmt.Errorf("parse input hcl: %s", diags.Error())
	}

	originalFile, originalDiags := hclwrite.ParseConfig(original, "original.tf", hcl.InitialPos)
	if originalDiags.HasErrors() {
		return nil, EditResult{}, fmt.Errorf("parse original hcl: %s", originalDiags.Error())
	}

	valueTokens, err := parseValueTokens(e.HCL)
	if err != nil {
		return nil, EditResult{}, err
	}

	leafFile := hclwrite.NewEmptyFile()
	leafFile.Body().SetAttributeRaw("value", valueTokens)
	leafValue, err := attributeExprValue(leafFile.Body().GetAttribute("value"), e.AttributePath)
	if err != nil {
		return nil, EditResult{}, err
	}

	targetBody := file.Body()
	createdTarget := false
	if e.TargetBlock != nil {
		originalTarget := findMatchingBlock(originalFile.Body(), *e.TargetBlock)
		target := targetBlockFromOriginal(file.Body(), originalFile.Body(), *e.TargetBlock)
		if target == nil {
			if e.Guard != nil && e.Guard.IfTargetExists {
				return data, EditResult{Changed: false, Occurrences: 0, Message: "guard skipped: target block missing"}, nil
			}

			if e.EnsureTargetBlock {
				target = ensureBlockPath(file.Body(), *e.TargetBlock)
				createdTarget = true
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

	rootName := path[0]
	existingRoot := targetBody.GetAttribute(rootName)
	if existingRoot == nil {
		if len(path) == 1 {
			targetBody.SetAttributeRaw(rootName, valueTokens)
		} else {
			targetBody.SetAttributeRaw(rootName, hclwrite.TokensForValue(objectPathValue(path[1:], leafValue)))
		}
		return file.Bytes(), EditResult{Changed: true, Occurrences: 1, Message: "insert hcl applied"}, nil
	}

	if len(path) == 1 {
		return data, EditResult{Changed: false, Occurrences: 0, Message: "attribute already exists"}, nil
	}

	rootValue, err := attributeExprValue(existingRoot, e.AttributePath)
	if err != nil {
		return nil, EditResult{}, err
	}

	updated, changed, exists, err := upsertObjectPath(rootValue, path[1:], leafValue, true, false)
	if err != nil {
		return nil, EditResult{}, err
	}
	if !changed {
		if exists {
			return data, EditResult{Changed: false, Occurrences: 0, Message: "attribute already exists"}, nil
		}
		if createdTarget {
			return file.Bytes(), EditResult{Changed: true, Occurrences: 1, Message: "insert hcl applied"}, nil
		}
		return data, EditResult{Changed: false, Occurrences: 0, Message: "no snippet entries found"}, nil
	}

	targetBody.SetAttributeRaw(rootName, hclwrite.TokensForValue(updated))
	return file.Bytes(), EditResult{Changed: true, Occurrences: 1, Message: "insert hcl applied"}, nil
}
