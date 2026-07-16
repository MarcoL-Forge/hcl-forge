package editor

import (
	"bytes"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

type SearchReplaceEdit struct {
	Old         string
	New         string
	TargetBlock *BlockSelector
	Attribute   string
	MatchMode   string
}

func (e SearchReplaceEdit) Apply(data []byte) ([]byte, EditResult, error) {
	if e.Old == "" {
		return nil, EditResult{}, fmt.Errorf("old value cannot be empty")
	}

	matcher, err := newDeleteMatcher(e.MatchMode)
	if err != nil {
		return nil, EditResult{}, err
	}

	if e.TargetBlock != nil && e.Attribute == "" {
		return nil, EditResult{}, fmt.Errorf("search_replace with block selector requires attribute")
	}

	if e.TargetBlock == nil && e.Attribute == "" {
		updatedBytes, occurrences, err := replaceBytes(data, e.Old, e.New, matcher.mode)
		if err != nil {
			return nil, EditResult{}, err
		}
		if occurrences == 0 {
			return data, EditResult{
				Changed:     false,
				Occurrences: 0,
				Message:     "no matches found",
			}, nil
		}

		return updatedBytes, EditResult{
			Changed:     true,
			Occurrences: occurrences,
			Message:     "search and replace applied",
		}, nil
	}

	file, diags := hclwrite.ParseConfig(data, "input.tf", hcl.InitialPos)
	if diags.HasErrors() {
		return nil, EditResult{}, fmt.Errorf("parse input hcl: %s", diags.Error())
	}

	var targetBodies []*hclwrite.Body
	if e.TargetBlock != nil {
		targetBodies = findMatchingBodies(file.Body(), *e.TargetBlock, matcher)
		if len(targetBodies) == 0 {
			return data, EditResult{Changed: false, Occurrences: 0, Message: "target block not found"}, nil
		}
	} else {
		targetBodies = collectAllBodies(file.Body())
	}

	occurrences := 0
	for _, body := range targetBodies {
		attr := body.GetAttribute(e.Attribute)
		if attr == nil {
			continue
		}

		tokens, replaced, err := replaceTokens(attr.Expr().BuildTokens(nil), e.Old, e.New, matcher.mode)
		if err != nil {
			return nil, EditResult{}, err
		}
		if replaced == 0 {
			continue
		}

		body.SetAttributeRaw(e.Attribute, tokens)
		occurrences += replaced
	}

	if occurrences == 0 {
		return data, EditResult{Changed: false, Occurrences: 0, Message: "no matches found"}, nil
	}

	return file.Bytes(), EditResult{
		Changed:     true,
		Occurrences: occurrences,
		Message:     "search and replace applied",
	}, nil
}

func replaceTokens(tokens hclwrite.Tokens, oldValue, newValue, mode string) (hclwrite.Tokens, int, error) {

	updated := make(hclwrite.Tokens, 0, len(tokens))
	occurrences := 0

	for _, token := range tokens {
		replacedBytes, count, err := replaceBytes(token.Bytes, oldValue, newValue, mode)
		if err != nil {
			return nil, 0, err
		}

		if count == 0 {
			updated = append(updated, &hclwrite.Token{Type: token.Type, Bytes: append([]byte(nil), token.Bytes...)})
			continue
		}

		updated = append(updated, &hclwrite.Token{Type: token.Type, Bytes: replacedBytes})
		occurrences += count
	}

	return updated, occurrences, nil
}

func replaceBytes(in []byte, oldValue, newValue, mode string) ([]byte, int, error) {
	if mode == "regex" {
		re, err := regexp.Compile(oldValue)
		if err != nil {
			return nil, 0, fmt.Errorf("invalid regex pattern %q: %w", oldValue, err)
		}

		matches := re.FindAllIndex(in, -1)
		if len(matches) == 0 {
			return append([]byte(nil), in...), 0, nil
		}

		out := re.ReplaceAll(in, []byte(newValue))
		return out, len(matches), nil
	}

	if mode == "glob" && hasGlobPattern(oldValue) {
		if _, err := filepath.Match(oldValue, ""); err != nil {
			return nil, 0, fmt.Errorf("invalid glob pattern %q: %w", oldValue, err)
		}

		re, err := regexp.Compile(globPatternToRegex(oldValue))
		if err != nil {
			return nil, 0, fmt.Errorf("invalid glob pattern %q: %w", oldValue, err)
		}

		matches := re.FindAllIndex(in, -1)
		if len(matches) == 0 {
			return append([]byte(nil), in...), 0, nil
		}

		out := re.ReplaceAll(in, []byte(newValue))
		return out, len(matches), nil
	}

	oldBytes := []byte(oldValue)
	newBytes := []byte(newValue)
	count := bytes.Count(in, oldBytes)
	if count == 0 {
		return append([]byte(nil), in...), 0, nil
	}

	return bytes.ReplaceAll(in, oldBytes, newBytes), count, nil
}

func globPatternToRegex(pattern string) string {
	escaped := regexp.QuoteMeta(pattern)
	escaped = strings.ReplaceAll(escaped, `\*`, `.*`)
	escaped = strings.ReplaceAll(escaped, `\?`, `.`)
	escaped = strings.ReplaceAll(escaped, `\[`, `[`) // preserve character classes
	escaped = strings.ReplaceAll(escaped, `\]`, `]`)
	return escaped
}
