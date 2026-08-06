package editor

import (
	"fmt"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

type SetAttributeHCLEdit struct {
	TargetBlock     *BlockSelector
	Attribute       string
	AttributePath   string
	ValueHCL        string
	CreateIfMissing bool
}

func (e SetAttributeHCLEdit) Apply(data []byte) ([]byte, EditResult, error) {
	return e.ApplyWithOriginal(data, data)
}

func (e SetAttributeHCLEdit) ApplyWithOriginal(data []byte, original []byte) ([]byte, EditResult, error) {
	if e.Attribute == "" && e.AttributePath == "" {
		return nil, EditResult{}, fmt.Errorf("set_attribute requires attribute or attribute_path")
	}
	if e.Attribute != "" && e.AttributePath != "" {
		return nil, EditResult{}, fmt.Errorf("set_attribute attribute and attribute_path are mutually exclusive")
	}
	if e.ValueHCL == "" {
		return nil, EditResult{}, fmt.Errorf("set_attribute requires value_hcl")
	}

	file, diags := hclwrite.ParseConfig(data, "input.tf", hcl.InitialPos)
	if diags.HasErrors() {
		return nil, EditResult{}, fmt.Errorf("parse input hcl: %s", diags.Error())
	}

	originalFile, originalDiags := hclwrite.ParseConfig(original, "original.tf", hcl.InitialPos)
	if originalDiags.HasErrors() {
		return nil, EditResult{}, fmt.Errorf("parse original hcl: %s", originalDiags.Error())
	}

	valueTokens, err := parseValueTokens(e.ValueHCL)
	if err != nil {
		return nil, EditResult{}, err
	}

	createIfMissing := e.CreateIfMissing

	targetBody := file.Body()
	if e.TargetBlock != nil {
		target := targetBlockFromOriginal(file.Body(), originalFile.Body(), *e.TargetBlock)
		if target == nil {
			return data, EditResult{Changed: false, Occurrences: 0, Message: "target block not found"}, nil
		}
		targetBody = target.Body()
	}

	if e.AttributePath != "" {
		path, err := parseAttributePath(e.AttributePath)
		if err != nil {
			return nil, EditResult{}, err
		}

		leafAttr := hclwrite.NewEmptyFile().Body()
		leafAttr.SetAttributeRaw("value", valueTokens)
		leafValue, err := attributeExprValue(leafAttr.GetAttribute("value"), e.AttributePath)
		if err != nil {
			return nil, EditResult{}, err
		}

		rootName := path[0]
		existingRoot := targetBody.GetAttribute(rootName)
		if existingRoot == nil {
			if !createIfMissing {
				return data, EditResult{Changed: false, Occurrences: 0, Message: "attribute path not found"}, nil
			}

			if len(path) == 1 {
				targetBody.SetAttributeRaw(rootName, valueTokens)
			} else {
				targetBody.SetAttributeRaw(rootName, hclwrite.TokensForValue(objectPathValue(path[1:], leafValue)))
			}

			return file.Bytes(), EditResult{Changed: true, Occurrences: 1, Message: "attribute created"}, nil
		}

		if len(path) == 1 {
			if tokensEqual(existingRoot.Expr().BuildTokens(nil), valueTokens) {
				return data, EditResult{Changed: false, Occurrences: 0, Message: "attribute already set"}, nil
			}

			targetBody.SetAttributeRaw(rootName, valueTokens)
			return file.Bytes(), EditResult{Changed: true, Occurrences: 1, Message: "attribute updated"}, nil
		}

		rootValue, err := attributeExprValue(existingRoot, e.AttributePath)
		if err != nil {
			return nil, EditResult{}, err
		}

		updated, changed, _, err := upsertObjectPath(rootValue, path[1:], leafValue, createIfMissing, true)
		if err != nil {
			return nil, EditResult{}, err
		}
		if !changed {
			return data, EditResult{Changed: false, Occurrences: 0, Message: "attribute path not found"}, nil
		}

		targetBody.SetAttributeRaw(rootName, hclwrite.TokensForValue(updated))
		return file.Bytes(), EditResult{Changed: true, Occurrences: 1, Message: "attribute updated"}, nil
	}

	existing := targetBody.GetAttribute(e.Attribute)
	if existing != nil {
		if tokensEqual(existing.Expr().BuildTokens(nil), valueTokens) {
			return data, EditResult{Changed: false, Occurrences: 0, Message: "attribute already set"}, nil
		}
		targetBody.SetAttributeRaw(e.Attribute, valueTokens)
		return file.Bytes(), EditResult{Changed: true, Occurrences: 1, Message: "attribute updated"}, nil
	}

	if !createIfMissing {
		return data, EditResult{Changed: false, Occurrences: 0, Message: "attribute not found"}, nil
	}

	targetBody.SetAttributeRaw(e.Attribute, valueTokens)
	return file.Bytes(), EditResult{Changed: true, Occurrences: 1, Message: "attribute created"}, nil
}

func parseValueTokens(valueHCL string) (hclwrite.Tokens, error) {
	snippet := []byte("value = " + valueHCL + "\n")
	snippetFile, snippetDiags := hclwrite.ParseConfig(snippet, "value.tf", hcl.InitialPos)
	if snippetDiags.HasErrors() {
		return nil, fmt.Errorf("parse value_hcl: %s", snippetDiags.Error())
	}

	attr := snippetFile.Body().GetAttribute("value")
	if attr == nil {
		return nil, fmt.Errorf("parse value_hcl: empty expression")
	}

	return attr.Expr().BuildTokens(nil), nil
}
