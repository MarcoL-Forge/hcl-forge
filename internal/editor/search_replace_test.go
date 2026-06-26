package editor

import "testing"

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
