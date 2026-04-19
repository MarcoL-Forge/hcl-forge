package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/Marc0l95/hcl-forge/internal/document"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

type ReplaceAttributeInput struct {
	Selector  string
	BlockType string
	Labels    []string
	Attribute string
	Value     string
	ValueType string
}

func ReplaceAttributeValue(doc *document.Document, input ReplaceAttributeInput) (*document.Document, error) {
	if doc == nil {
		return nil, fmt.Errorf("document is nil")
	}

	parsedFile, diags := hclwrite.ParseConfig(doc.Raw, doc.Path, hcl.InitialPos)
	if diags.HasErrors() {
		return nil, fmt.Errorf("parse config: %s", diags.Error())
	}

	targetBody, attributeName, err := resolveAttributeTarget(parsedFile.Body(), input)
	if err != nil {
		return nil, err
	}

	value, rawTokens, useRaw, err := parseAttributeValue(input.Value, input.ValueType)
	if err != nil {
		return nil, err
	}

	if useRaw {
		targetBody.SetAttributeRaw(attributeName, rawTokens)
	} else {
		targetBody.SetAttributeValue(attributeName, value)
	}

	return &document.Document{
		Path: doc.Path,
		Raw:  parsedFile.Bytes(),
	}, nil
}

func resolveAttributeTarget(body *hclwrite.Body, input ReplaceAttributeInput) (*hclwrite.Body, string, error) {
	if strings.TrimSpace(input.Selector) != "" {
		return resolveSelectorTarget(body, input.Selector)
	}

	if input.Attribute == "" {
		return nil, "", fmt.Errorf("attribute is empty")
	}

	if input.BlockType == "" {
		return body, input.Attribute, nil
	}

	block := findBlock(body.Blocks(), input.BlockType, input.Labels)
	if block == nil {
		return nil, "", fmt.Errorf("block %s with labels %v not found", input.BlockType, input.Labels)
	}

	return block.Body(), input.Attribute, nil
}

func resolveSelectorTarget(body *hclwrite.Body, selector string) (*hclwrite.Body, string, error) {
	segments := splitSelector(selector)
	if len(segments) == 0 {
		return nil, "", fmt.Errorf("selector is empty")
	}

	if len(segments) == 1 {
		return body, segments[0], nil
	}

	attributeName := segments[len(segments)-1]

	var matchedBody *hclwrite.Body
	matchCount := 0

	for labelCount := len(segments) - 2; labelCount >= 0; labelCount-- {
		labels := segments[1 : 1+labelCount]
		nestedTypes := segments[1+labelCount : len(segments)-1]

		block := findBlock(body.Blocks(), segments[0], labels)
		if block == nil {
			continue
		}

		currentBody := block.Body()
		resolved := true
		for _, nestedType := range nestedTypes {
			nestedBlock := findBlock(currentBody.Blocks(), nestedType, nil)
			if nestedBlock == nil {
				resolved = false
				break
			}

			currentBody = nestedBlock.Body()
		}

		if !resolved {
			continue
		}

		matchedBody = currentBody
		matchCount++
	}

	if matchCount == 0 {
		return nil, "", fmt.Errorf("selector %q not found", selector)
	}

	if matchCount > 1 {
		return nil, "", fmt.Errorf("selector %q is ambiguous", selector)
	}

	return matchedBody, attributeName, nil
}

func splitSelector(raw string) []string {
	parts := strings.Split(raw, ".")
	segments := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			segments = append(segments, trimmed)
		}
	}

	return segments
}

func findBlock(blocks []*hclwrite.Block, blockType string, labels []string) *hclwrite.Block {
	for _, block := range blocks {
		if block.Type() != blockType {
			continue
		}

		if equalLabels(block.Labels(), labels) {
			return block
		}
	}

	return nil
}

func equalLabels(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}

	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}

	return true
}

func parseAttributeValue(raw string, valueType string) (cty.Value, hclwrite.Tokens, bool, error) {
	switch strings.ToLower(strings.TrimSpace(valueType)) {
	case "hcl", "expr", "expression", "raw":
		tokens, err := parseExpressionTokens(raw)
		if err != nil {
			return cty.NilVal, nil, false, err
		}

		return cty.NilVal, tokens, true, nil
	}

	value, err := parseCtyValue(raw, valueType)
	if err != nil {
		return cty.NilVal, nil, false, err
	}

	return value, nil, false, nil
}

func parseCtyValue(raw string, valueType string) (cty.Value, error) {
	switch strings.ToLower(strings.TrimSpace(valueType)) {
	case "", "string":
		return cty.StringVal(raw), nil
	case "bool", "boolean":
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			return cty.NilVal, fmt.Errorf("parse bool value: %w", err)
		}
		return cty.BoolVal(parsed), nil
	case "number", "int", "integer":
		if parsedInt, err := strconv.ParseInt(raw, 10, 64); err == nil {
			return cty.NumberIntVal(parsedInt), nil
		}

		parsedFloat, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return cty.NilVal, fmt.Errorf("parse number value: %w", err)
		}

		return cty.NumberFloatVal(parsedFloat), nil
	default:
		return cty.NilVal, fmt.Errorf("unsupported value type %q", valueType)
	}
}

func parseExpressionTokens(raw string) (hclwrite.Tokens, error) {
	snippet := []byte("value = " + raw + "\n")
	parsedFile, diags := hclwrite.ParseConfig(snippet, "inline.hcl", hcl.InitialPos)
	if diags.HasErrors() {
		return nil, fmt.Errorf("parse hcl expression: %s", diags.Error())
	}

	attribute := parsedFile.Body().GetAttribute("value")
	if attribute == nil {
		return nil, fmt.Errorf("parse hcl expression: value attribute not found")
	}

	return attribute.Expr().BuildTokens(nil), nil
}
