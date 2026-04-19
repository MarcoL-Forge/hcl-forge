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

	if input.BlockType == "" {
		return nil, fmt.Errorf("block type is empty")
	}

	if input.Attribute == "" {
		return nil, fmt.Errorf("attribute is empty")
	}

	parsedFile, diags := hclwrite.ParseConfig(doc.Raw, doc.Path, hcl.InitialPos)
	if diags.HasErrors() {
		return nil, fmt.Errorf("parse config: %s", diags.Error())
	}

	block := findBlock(parsedFile.Body().Blocks(), input.BlockType, input.Labels)
	if block == nil {
		return nil, fmt.Errorf("block %s with labels %v not found", input.BlockType, input.Labels)
	}

	value, err := parseCtyValue(input.Value, input.ValueType)
	if err != nil {
		return nil, err
	}

	block.Body().SetAttributeValue(input.Attribute, value)

	return &document.Document{
		Path: doc.Path,
		Raw:  parsedFile.Bytes(),
	}, nil
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
