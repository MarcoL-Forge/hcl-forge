package editor

import (
	"strings"
	"testing"
)

func TestSearchReplaceEdit_Apply_ReplacesAllOccurrences(t *testing.T) {
	edit := SearchReplaceEdit{Old: "foo", New: "bar"}

	updated, result, err := edit.Apply([]byte("foo baz foo"))
	if err != nil {
		t.Fatalf("apply edit: %v", err)
	}

	if string(updated) != "bar baz bar" {
		t.Fatalf("unexpected output: %q", string(updated))
	}

	if !result.Changed {
		t.Fatalf("expected Changed=true")
	}

	if result.Occurrences != 2 {
		t.Fatalf("expected 2 occurrences, got %d", result.Occurrences)
	}
}

func TestSearchReplaceEdit_Apply_NoMatch(t *testing.T) {
	edit := SearchReplaceEdit{Old: "missing", New: "bar"}

	updated, result, err := edit.Apply([]byte("foo baz foo"))
	if err != nil {
		t.Fatalf("apply edit: %v", err)
	}

	if string(updated) != "foo baz foo" {
		t.Fatalf("content changed unexpectedly: %q", string(updated))
	}

	if result.Changed {
		t.Fatalf("expected Changed=false")
	}

	if result.Occurrences != 0 {
		t.Fatalf("expected 0 occurrences, got %d", result.Occurrences)
	}
}

func TestSearchReplaceEdit_Apply_EmptyOldValue(t *testing.T) {
	edit := SearchReplaceEdit{Old: "", New: "bar"}

	_, _, err := edit.Apply([]byte("foo"))
	if err == nil {
		t.Fatalf("expected error for empty old value")
	}
}

func TestSearchReplaceEdit_Apply_TargetedByPathAndAttribute(t *testing.T) {
	input := `module "tfe_workspace" "example1" {
  name = "example-rtl-int-workspace1-gke"
}

module "tfe_workspace" "example2" {
  name = "example-rtl-int-workspace2-prj"
}
`

	edit := SearchReplaceEdit{
		Old: "rtl-int-",
		New: "prod-",
		TargetBlock: &BlockSelector{
			Type:   "module",
			Labels: []string{"tfe_workspace", "example2"},
		},
		Attribute: "name",
	}

	updated, result, err := edit.Apply([]byte(input))
	if err != nil {
		t.Fatalf("apply targeted search_replace: %v", err)
	}

	out := string(updated)
	if !result.Changed || result.Occurrences != 1 {
		t.Fatalf("expected one targeted replacement, got changed=%v occurrences=%d", result.Changed, result.Occurrences)
	}

	if !strings.Contains(out, `name = "example-prod-workspace2-prj"`) {
		t.Fatalf("expected example2 name to be updated, got:\n%s", out)
	}

	if !strings.Contains(out, `name = "example-rtl-int-workspace1-gke"`) {
		t.Fatalf("expected example1 name to remain unchanged, got:\n%s", out)
	}
}

func TestSearchReplaceEdit_Apply_TargetBlockNotFound_IsNoOp(t *testing.T) {
	input := `module "tfe_workspace" "example1" {
  name = "example-rtl-int-workspace1-gke"
}
`

	edit := SearchReplaceEdit{
		Old: "rtl-int-",
		New: "prod-",
		TargetBlock: &BlockSelector{
			Type:   "module",
			Labels: []string{"tfe_workspace", "missing"},
		},
		Attribute: "name",
	}

	updated, result, err := edit.Apply([]byte(input))
	if err != nil {
		t.Fatalf("expected no error for missing target block, got: %v", err)
	}

	if result.Changed || result.Occurrences != 0 {
		t.Fatalf("expected no change for missing target block, got changed=%v occurrences=%d", result.Changed, result.Occurrences)
	}

	if string(updated) != input {
		t.Fatalf("expected output to remain unchanged")
	}
}

func TestSearchReplaceEdit_Apply_RegexAcrossMatchingWorkspaces(t *testing.T) {
	input := `module "tfe_workspace" "example1" {
  name = "example-rtl-int-workspace1-gke01"
}

module "tfe_workspace" "example2" {
  name = "example-rtl-int-workspace2-prj"
}

module "tfe_workspace" "example3" {
  name = "example-rtl-int-workspace3-sm"
}
`

	edit := SearchReplaceEdit{
		Old: "rtl-int-|01",
		New: "",
		TargetBlock: &BlockSelector{
			Type:   "module",
			Labels: []string{"tfe_workspace", "example.*"},
		},
		Attribute: "name",
		MatchMode: "regex",
	}

	updated, result, err := edit.Apply([]byte(input))
	if err != nil {
		t.Fatalf("apply regex scoped search_replace: %v", err)
	}

	out := string(updated)
	if !result.Changed || result.Occurrences != 4 {
		t.Fatalf("expected four regex replacements, got changed=%v occurrences=%d", result.Changed, result.Occurrences)
	}

	if !strings.Contains(out, `name = "example-workspace1-gke"`) {
		t.Fatalf("expected example1 name stripped of rtl-int-/01, got:\n%s", out)
	}

	if !strings.Contains(out, `name = "example-workspace2-prj"`) {
		t.Fatalf("expected example2 name stripped of rtl-int-, got:\n%s", out)
	}

	if !strings.Contains(out, `name = "example-workspace3-sm"`) {
		t.Fatalf("expected example3 name stripped of rtl-int-, got:\n%s", out)
	}
}

func TestSearchReplaceEdit_Apply_InvalidRegex(t *testing.T) {
	edit := SearchReplaceEdit{
		Old:       "(",
		New:       "x",
		MatchMode: "regex",
	}

	_, _, err := edit.Apply([]byte("foo"))
	if err == nil {
		t.Fatalf("expected invalid regex error")
	}
}

func TestSearchReplaceEdit_Apply_GlobAcrossMatchingWorkspaces(t *testing.T) {
	input := `module "tfe_workspace" "example1" {
  name = "example-rtl-int-workspace1-gke01"
}

module "tfe_workspace" "example2" {
  name = "example-rtl-int-workspace2-prj"
}

module "tfe_workspace" "example3" {
  name = "example-rtl-int-workspace3-sm"
}
`

	edit := SearchReplaceEdit{
		Old: "rtl-int-*",
		New: "prod-",
		TargetBlock: &BlockSelector{
			Type:   "module",
			Labels: []string{"tfe_workspace", "example*"},
		},
		Attribute: "name",
		MatchMode: "glob",
	}

	updated, result, err := edit.Apply([]byte(input))
	if err != nil {
		t.Fatalf("apply glob scoped search_replace: %v", err)
	}

	out := string(updated)
	if !result.Changed || result.Occurrences != 3 {
		t.Fatalf("expected three glob replacements, got changed=%v occurrences=%d", result.Changed, result.Occurrences)
	}

	if !strings.Contains(out, `name = "example-prod-"`) {
		t.Fatalf("expected glob replacement to apply, got:\n%s", out)
	}
}

func TestSearchReplaceEdit_Apply_InvalidGlob(t *testing.T) {
	edit := SearchReplaceEdit{
		Old:       "[",
		New:       "x",
		MatchMode: "glob",
	}

	_, _, err := edit.Apply([]byte("foo"))
	if err == nil {
		t.Fatalf("expected invalid glob error")
	}
}

func TestApplyEdits_SearchReplaceScopedUsesOriginalSelectorSnapshot(t *testing.T) {
	input := []byte(`module "tfe_workspace" "example2" {
  name = "example-rtl-int-workspace2-prj"
}
`)

	edits := []Edit{
		SearchReplaceEdit{Old: "example2", New: "example2prd"},
		SearchReplaceEdit{
			Old: "rtl-int-",
			New: "prd-",
			TargetBlock: &BlockSelector{
				Type:   "module",
				Labels: []string{"tfe_workspace", "example2"},
			},
			Attribute: "name",
		},
	}

	updated, results, err := ApplyEdits(input, edits)
	if err != nil {
		t.Fatalf("apply edits: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 edit results, got %d", len(results))
	}

	out := string(updated)
	if !strings.Contains(out, `module "tfe_workspace" "example2prd"`) {
		t.Fatalf("expected first edit to rename module label, got:\n%s", out)
	}

	if !strings.Contains(out, `name = "example-prd-workspace2-prj"`) {
		t.Fatalf("expected second edit to target old selector and update name, got:\n%s", out)
	}
}
