package editor

import (
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

func parseAttributePath(path string) ([]string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return nil, fmt.Errorf("attribute_path cannot be empty")
	}

	parts := strings.Split(trimmed, ".")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
		if parts[i] == "" {
			return nil, fmt.Errorf("attribute_path contains empty segment")
		}
	}

	return parts, nil
}

func attributeExprValue(attr *hclwrite.Attribute, path string) (cty.Value, error) {
	exprSource := attr.Expr().BuildTokens(nil).Bytes()
	expr, parseDiags := hclsyntax.ParseExpression(exprSource, "attribute_path.hcl", hcl.Pos{Line: 1, Column: 1})
	if parseDiags.HasErrors() {
		return cty.NilVal, fmt.Errorf("parse attribute_path expression %q: %s", path, parseDiags.Error())
	}

	value, evalDiags := expr.Value(nil)
	if evalDiags.HasErrors() {
		return cty.NilVal, fmt.Errorf("attribute_path %q requires literal expression: %s", path, evalDiags.Error())
	}

	return value, nil
}

func objectPathValue(path []string, leaf cty.Value) cty.Value {
	if len(path) == 0 {
		return leaf
	}

	current := leaf
	for i := len(path) - 1; i >= 0; i-- {
		current = cty.ObjectVal(map[string]cty.Value{path[i]: current})
	}

	return current
}

func upsertObjectPath(value cty.Value, path []string, leaf cty.Value, createMissing bool, overwrite bool) (cty.Value, bool, bool, error) {
	if len(path) == 0 {
		return value, false, false, nil
	}

	if !(value.Type().IsObjectType() || value.Type().IsMapType()) {
		return cty.NilVal, false, false, fmt.Errorf("attribute_path traverses non-object value")
	}

	items := value.AsValueMap()
	segment := path[0]

	if len(path) == 1 {
		existing, exists := items[segment]
		if exists {
			if !overwrite {
				return value, false, true, nil
			}
			if existing.RawEquals(leaf) {
				return value, false, true, nil
			}
		}

		if !exists && !createMissing {
			return value, false, false, nil
		}

		items[segment] = leaf
		return rebuildObjectLike(value, items), true, exists, nil
	}

	child, exists := items[segment]
	if !exists {
		if !createMissing {
			return value, false, false, nil
		}

		createdChild := objectPathValue(path[1:], leaf)
		items[segment] = createdChild
		return rebuildObjectLike(value, items), true, false, nil
	}

	updatedChild, changed, leafExisted, err := upsertObjectPath(child, path[1:], leaf, createMissing, overwrite)
	if err != nil {
		return cty.NilVal, false, false, err
	}
	if !changed {
		return value, false, leafExisted, nil
	}

	items[segment] = updatedChild
	return rebuildObjectLike(value, items), true, leafExisted, nil
}
