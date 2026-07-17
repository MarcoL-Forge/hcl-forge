package editor

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

type DeleteHCLEdit struct {
	TargetBlock *BlockSelector
	Attribute   string
	DeleteAll   bool
	KeepOnly    bool
	MatchMode   string
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
