package editor

import (
	"bytes"
	"fmt"
)

type SearchReplaceEdit struct {
	Old string
	New string
}

func (e SearchReplaceEdit) Apply(data []byte) ([]byte, EditResult, error) {
	if e.Old == "" {
		return nil, EditResult{}, fmt.Errorf("old value cannot be empty")
	}

	oldBytes := []byte(e.Old)
	newBytes := []byte(e.New)

	occurrences := bytes.Count(data, oldBytes)
	if occurrences == 0 {
		return data, EditResult{
			Changed:     false,
			Occurrences: 0,
			Message:     "no matches found",
		}, nil
	}

	updated := bytes.ReplaceAll(data, oldBytes, newBytes)

	return updated, EditResult{
		Changed:     true,
		Occurrences: occurrences,
		Message:     "search and replace applied",
	}, nil
}
