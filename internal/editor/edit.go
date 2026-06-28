package editor

type EditResult struct {
	Changed     bool
	Occurrences int
	Message     string
}

type Edit interface {
	Apply(data []byte) ([]byte, EditResult, error)
}

func ApplyEdits(data []byte, edits []Edit) ([]byte, []EditResult, error) {
	current := data
	results := make([]EditResult, 0, len(edits))

	for _, edit := range edits {
		updated, result, err := edit.Apply(current)
		if err != nil {
			return nil, results, err
		}

		current = updated
		results = append(results, result)
	}

	return current, results, nil
}

func HasChanges(results []EditResult) bool {
	for _, result := range results {
		if result.Changed {
			return true
		}
	}

	return false
}
