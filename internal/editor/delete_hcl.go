package editor

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

type DeleteHCLEdit struct {
	TargetBlock   *BlockSelector
	Attribute     string
	AttributePath string
	ElementKey    string
	ElementMatch  string
	DeleteAll     bool
	KeepOnly      bool
	MatchMode     string
}

func (e DeleteHCLEdit) Apply(data []byte) ([]byte, EditResult, error) {
	return e.ApplyWithOriginal(data, data)
}

func (e DeleteHCLEdit) ApplyWithOriginal(data []byte, original []byte) ([]byte, EditResult, error) {
	matcher, err := newDeleteMatcher(e.MatchMode)
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

	if e.KeepOnly {
		if e.TargetBlock == nil {
			if e.AttributePath == "" || e.ElementKey == "" || e.ElementMatch == "" {
				return nil, EditResult{}, fmt.Errorf("keep_only requires block selector or attribute_path list filter")
			}
		} else {
			removed := keepOnlyMatchingBlocksByOriginal(file.Body(), originalFile.Body(), *e.TargetBlock, matcher)
			if removed == 0 {
				return data, EditResult{Changed: false, Occurrences: 0, Message: "no blocks removed by keep_only"}, nil
			}

			return file.Bytes(), EditResult{Changed: true, Occurrences: removed, Message: "non-matching blocks removed by keep_only"}, nil
		}
	}

	if e.AttributePath != "" {
		targetBodies := []*hclwrite.Body{file.Body()}
		if e.TargetBlock != nil {
			positions := findMatchingBodyPositions(originalFile.Body(), *e.TargetBlock, matcher)
			if !e.DeleteAll && len(positions) > 1 {
				positions = positions[:1]
			}

			targetBodies = make([]*hclwrite.Body, 0, len(positions))
			for _, position := range positions {
				targetBody := bodyAtPosition(file.Body(), position)
				if targetBody != nil {
					targetBodies = append(targetBodies, targetBody)
				}
			}

			if len(targetBodies) == 0 {
				return data, EditResult{Changed: false, Occurrences: 0, Message: "target block not found"}, nil
			}
		} else if e.DeleteAll {
			targetBodies = collectAllBodies(file.Body())
		}

		deleted := 0
		for _, body := range targetBodies {
			removed, err := deleteByAttributePath(body, e, matcher)
			if err != nil {
				return nil, EditResult{}, err
			}

			if removed == 0 {
				continue
			}

			deleted += removed
			if !e.DeleteAll {
				break
			}
		}

		if deleted == 0 {
			return data, EditResult{Changed: false, Occurrences: 0, Message: "attribute path not found"}, nil
		}

		return file.Bytes(), EditResult{Changed: true, Occurrences: deleted, Message: "attribute path updated"}, nil
	}

	if e.KeepOnly {
		if e.TargetBlock == nil {
			return nil, EditResult{}, fmt.Errorf("keep_only requires block selector")
		}

		removed := keepOnlyMatchingBlocksByOriginal(file.Body(), originalFile.Body(), *e.TargetBlock, matcher)
		if removed == 0 {
			return data, EditResult{Changed: false, Occurrences: 0, Message: "no blocks removed by keep_only"}, nil
		}

		return file.Bytes(), EditResult{Changed: true, Occurrences: removed, Message: "non-matching blocks removed by keep_only"}, nil
	}

	if e.Attribute != "" {
		targetBodies := []*hclwrite.Body{file.Body()}
		if e.TargetBlock != nil {
			positions := findMatchingBodyPositions(originalFile.Body(), *e.TargetBlock, matcher)
			if !e.DeleteAll && len(positions) > 1 {
				positions = positions[:1]
			}

			targetBodies = make([]*hclwrite.Body, 0, len(positions))
			for _, position := range positions {
				targetBody := bodyAtPosition(file.Body(), position)
				if targetBody != nil {
					targetBodies = append(targetBodies, targetBody)
				}
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
					if !matcher.matches(e.Attribute, name) {
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

	positions := findMatchingBodyPositions(originalFile.Body(), *e.TargetBlock, matcher)
	if len(positions) == 0 {
		return data, EditResult{Changed: false, Occurrences: 0, Message: "block not found"}, nil
	}

	if !e.DeleteAll {
		positions = positions[:1]
	} else {
		sortBodyPositionsForDeletion(positions)
	}

	removed := 0
	for _, position := range positions {
		if removeBlockAtPosition(file.Body(), position) {
			removed++
		}
	}

	if removed == 0 {
		return data, EditResult{Changed: false, Occurrences: 0, Message: "block not found"}, nil
	}

	return file.Bytes(), EditResult{Changed: true, Occurrences: removed, Message: "block deleted"}, nil
}

func deleteByAttributePath(body *hclwrite.Body, edit DeleteHCLEdit, matcher deleteMatcher) (int, error) {
	path, err := parseAttributePath(edit.AttributePath)
	if err != nil {
		return 0, err
	}

	root := path[0]
	attr := body.GetAttribute(root)
	if attr == nil {
		return 0, nil
	}

	if len(path) == 1 && edit.ElementKey == "" && !edit.KeepOnly {
		body.RemoveAttribute(root)
		return 1, nil
	}

	value, err := attributeExprValue(attr, edit.AttributePath)
	if err != nil {
		return 0, err
	}

	var (
		updated cty.Value
		changed bool
		count   int
		opErr   error
	)

	if edit.ElementKey != "" {
		updated, changed, count, opErr = filterListAtPath(value, path[1:], edit.ElementKey, edit.ElementMatch, edit.KeepOnly, matcher)
	} else {
		updated, changed, count, opErr = deleteNestedKeyAtPath(value, path[1:], edit.KeepOnly, edit.DeleteAll, matcher)
	}
	if opErr != nil {
		return 0, opErr
	}
	if !changed {
		return 0, nil
	}

	body.SetAttributeRaw(root, hclwrite.TokensForValue(updated))
	return count, nil
}

func deleteNestedKeyAtPath(value cty.Value, path []string, keepOnly bool, deleteAll bool, matcher deleteMatcher) (cty.Value, bool, int, error) {
	if len(path) == 0 {
		return value, false, 0, nil
	}

	if !(value.Type().IsObjectType() || value.Type().IsMapType()) {
		return value, false, 0, nil
	}

	items := value.AsValueMap()
	segment := path[0]

	if len(path) == 1 {
		updated := make(map[string]cty.Value, len(items))
		changed := false
		count := 0

		for key, item := range items {
			matches := matcher.matches(segment, key)
			if keepOnly {
				if matches {
					updated[key] = item
				} else {
					changed = true
					count++
				}
				continue
			}

			if matches {
				changed = true
				count++
				if !deleteAll {
					for k, v := range items {
						if k == key {
							continue
						}
						updated[k] = v
					}
					return rebuildObjectLike(value, updated), true, 1, nil
				}
				continue
			}

			updated[key] = item
		}

		if !changed {
			return value, false, 0, nil
		}

		return rebuildObjectLike(value, updated), true, count, nil
	}

	child, ok := items[segment]
	if !ok {
		return value, false, 0, nil
	}

	updatedChild, changed, count, err := deleteNestedKeyAtPath(child, path[1:], keepOnly, deleteAll, matcher)
	if err != nil {
		return cty.NilVal, false, 0, err
	}
	if !changed {
		return value, false, 0, nil
	}

	items[segment] = updatedChild
	return rebuildObjectLike(value, items), true, count, nil
}

func filterListAtPath(value cty.Value, path []string, key string, match string, keepOnly bool, matcher deleteMatcher) (cty.Value, bool, int, error) {
	if len(path) > 0 {
		if !(value.Type().IsObjectType() || value.Type().IsMapType()) {
			return value, false, 0, nil
		}

		items := value.AsValueMap()
		child, ok := items[path[0]]
		if !ok {
			return value, false, 0, nil
		}

		updatedChild, changed, count, err := filterListAtPath(child, path[1:], key, match, keepOnly, matcher)
		if err != nil {
			return cty.NilVal, false, 0, err
		}
		if !changed {
			return value, false, 0, nil
		}

		items[path[0]] = updatedChild
		return rebuildObjectLike(value, items), true, count, nil
	}

	if !(value.Type().IsTupleType() || value.Type().IsListType() || value.Type().IsSetType()) {
		return value, false, 0, nil
	}

	elements := value.AsValueSlice()
	filtered := make([]cty.Value, 0, len(elements))
	removed := 0

	for _, elem := range elements {
		matches := false
		if elem.Type().IsObjectType() || elem.Type().IsMapType() {
			obj := elem.AsValueMap()
			if candidate, ok := obj[key]; ok {
				if str, ok := ctyString(candidate); ok {
					matches = matcher.matches(match, str)
				}
			}
		}

		if keepOnly {
			if matches {
				filtered = append(filtered, elem)
			} else {
				removed++
			}
			continue
		}

		if matches {
			removed++
			continue
		}

		filtered = append(filtered, elem)
	}

	if removed == 0 {
		return value, false, 0, nil
	}

	return rebuildSequenceLike(value, filtered), true, removed, nil
}

func ctyString(value cty.Value) (string, bool) {
	if !value.IsKnown() || value.IsNull() {
		return "", false
	}

	if value.Type() == cty.String {
		return value.AsString(), true
	}

	return "", false
}

func rebuildObjectLike(original cty.Value, items map[string]cty.Value) cty.Value {
	if original.Type().IsMapType() {
		if len(items) == 0 {
			return cty.MapValEmpty(original.Type().ElementType())
		}

		return cty.MapVal(items)
	}

	return cty.ObjectVal(items)
}

func rebuildSequenceLike(original cty.Value, elems []cty.Value) cty.Value {
	if original.Type().IsListType() {
		if len(elems) == 0 {
			return cty.ListValEmpty(original.Type().ElementType())
		}

		return cty.ListVal(elems)
	}

	if original.Type().IsSetType() {
		if len(elems) == 0 {
			return cty.SetValEmpty(original.Type().ElementType())
		}

		return cty.SetVal(elems)
	}

	return cty.TupleVal(elems)
}

func keepOnlyMatchingBlocksByOriginal(currentRoot, originalRoot *hclwrite.Body, selector BlockSelector, matcher deleteMatcher) int {
	positions := findKeepOnlyRemovalPositions(originalRoot, selector, matcher)
	if len(positions) == 0 {
		return 0
	}

	sortBodyPositionsForDeletion(positions)
	removed := 0
	for _, position := range positions {
		if removeBlockAtPosition(currentRoot, position) {
			removed++
		}
	}

	return removed
}

func findKeepOnlyRemovalPositions(root *hclwrite.Body, selector BlockSelector, matcher deleteMatcher) []bodyPosition {
	return findKeepOnlyRemovalPositionsWithParents(root, selector, nil, nil, matcher)
}

func findKeepOnlyRemovalPositionsWithParents(
	body *hclwrite.Body,
	selector BlockSelector,
	ancestry []ParentSelector,
	path bodyPosition,
	matcher deleteMatcher,
) []bodyPosition {
	positions := make([]bodyPosition, 0)

	for index, block := range body.Blocks() {
		nextPath := append(append(bodyPosition(nil), path...), index)
		nextAncestry := append(append([]ParentSelector(nil), ancestry...), ParentSelector{
			Type:   block.Type(),
			Labels: append([]string(nil), block.Labels()...),
		})

		positions = append(positions, findKeepOnlyRemovalPositionsWithParents(block.Body(), selector, nextAncestry, nextPath, matcher)...)

		if !blockInKeepOnlyScope(block, selector, ancestry, matcher) {
			continue
		}

		if blockMatchesDeleteSelector(block, selector, matcher) {
			continue
		}

		positions = append(positions, nextPath)
	}

	return positions
}

func sortBodyPositionsForDeletion(positions []bodyPosition) {
	sort.Slice(positions, func(i, j int) bool {
		a := positions[i]
		b := positions[j]

		if len(a) != len(b) {
			return len(a) > len(b)
		}

		for k := 0; k < len(a); k++ {
			if a[k] == b[k] {
				continue
			}

			return a[k] > b[k]
		}

		return false
	})
}

func removeBlockAtPosition(root *hclwrite.Body, position bodyPosition) bool {
	if len(position) == 0 {
		return false
	}

	parent := root
	for i := 0; i < len(position)-1; i++ {
		blocks := parent.Blocks()
		idx := position[i]
		if idx < 0 || idx >= len(blocks) {
			return false
		}

		parent = blocks[idx].Body()
	}

	blocks := parent.Blocks()
	idx := position[len(position)-1]
	if idx < 0 || idx >= len(blocks) {
		return false
	}

	parent.RemoveBlock(blocks[idx])
	return true
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

func blockMatchesDeleteSelector(block *hclwrite.Block, selector BlockSelector, matcher deleteMatcher) bool {
	if !matcher.matches(selector.Type, block.Type()) {
		return false
	}

	labels := block.Labels()
	if len(selector.Labels) != len(labels) {
		return false
	}

	for i := range labels {
		if !matcher.matches(selector.Labels[i], labels[i]) {
			return false
		}
	}

	return true
}

func parentsMatchDeleteSelector(ancestry, expected []ParentSelector, matcher deleteMatcher) bool {
	if len(expected) == 0 {
		return true
	}

	if len(ancestry) != len(expected) {
		return false
	}

	for i := range ancestry {
		if !matcher.matches(expected[i].Type, ancestry[i].Type) {
			return false
		}

		if len(ancestry[i].Labels) != len(expected[i].Labels) {
			return false
		}

		for j := range ancestry[i].Labels {
			if !matcher.matches(expected[i].Labels[j], ancestry[i].Labels[j]) {
				return false
			}
		}
	}

	return true
}

func blockInKeepOnlyScope(block *hclwrite.Block, selector BlockSelector, ancestry []ParentSelector, matcher deleteMatcher) bool {
	if !parentsMatchKeepOnlyScope(ancestry, selector.Parents, matcher) {
		return false
	}

	if !matchesKeepOnlyScopeValue(selector.Type, block.Type(), matcher.mode) {
		return false
	}

	labels := block.Labels()
	if len(selector.Labels) != len(labels) {
		return false
	}

	for i := range labels {
		if !matchesKeepOnlyScopeValue(selector.Labels[i], labels[i], matcher.mode) {
			return false
		}
	}

	return true
}

func parentsMatchKeepOnlyScope(ancestry, expected []ParentSelector, matcher deleteMatcher) bool {
	if len(expected) == 0 {
		return true
	}

	if len(ancestry) != len(expected) {
		return false
	}

	for i := range ancestry {
		if !matchesKeepOnlyScopeValue(expected[i].Type, ancestry[i].Type, matcher.mode) {
			return false
		}

		if len(ancestry[i].Labels) != len(expected[i].Labels) {
			return false
		}

		for j := range ancestry[i].Labels {
			if !matchesKeepOnlyScopeValue(expected[i].Labels[j], ancestry[i].Labels[j], matcher.mode) {
				return false
			}
		}
	}

	return true
}

func matchesKeepOnlyScopeValue(pattern, value, mode string) bool {
	if isPatternExpression(pattern, mode) {
		return true
	}

	return pattern == value
}

func isPatternExpression(pattern, mode string) bool {
	if mode == "regex" {
		return strings.ContainsAny(pattern, `.*+?()[]{}|^$\\`)
	}

	return hasGlobPattern(pattern)
}

type deleteMatcher struct {
	mode  string
	cache map[string]*regexp.Regexp
}

func newDeleteMatcher(mode string) (deleteMatcher, error) {
	switch mode {
	case "", "glob":
		return deleteMatcher{mode: "glob"}, nil
	case "regex":
		return deleteMatcher{mode: "regex", cache: map[string]*regexp.Regexp{}}, nil
	default:
		return deleteMatcher{}, fmt.Errorf("unsupported match_mode %q", mode)
	}
}

func (m *deleteMatcher) compileRegex(pattern string) (*regexp.Regexp, error) {
	if m.cache == nil {
		m.cache = map[string]*regexp.Regexp{}
	}

	if compiled, ok := m.cache[pattern]; ok {
		return compiled, nil
	}

	compiled, err := regexp.Compile("^(?:" + pattern + ")$")
	if err != nil {
		return nil, err
	}

	m.cache[pattern] = compiled
	return compiled, nil
}

func (m *deleteMatcher) matches(pattern, value string) bool {
	if m.mode == "regex" {
		compiled, err := m.compileRegex(pattern)
		if err != nil {
			return false
		}

		return compiled.MatchString(value)
	}

	return matchesPattern(pattern, value)
}
