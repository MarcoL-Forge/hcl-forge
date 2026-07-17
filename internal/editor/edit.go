package editor

import (
	"fmt"
	"time"

	"github.com/MarcoL-Forge/hcl-forge/internal/logging"
)

type EditResult struct {
	Changed     bool
	Occurrences int
	Message     string
}

type Edit interface {
	Apply(data []byte) ([]byte, EditResult, error)
}

type OriginalAwareEdit interface {
	ApplyWithOriginal(data []byte, original []byte) ([]byte, EditResult, error)
}

func ApplyEdits(data []byte, edits []Edit) ([]byte, []EditResult, error) {
	current := data
	results := make([]EditResult, 0, len(edits))
	logger := logging.Default()

	for i, edit := range edits {
		editType := fmt.Sprintf("%T", edit)
		start := time.Now()
		logger.Debug("edit_start", map[string]any{"index": i, "type": editType})

		var (
			updated []byte
			result  EditResult
			err     error
		)

		if originalAwareEdit, ok := edit.(OriginalAwareEdit); ok {
			updated, result, err = originalAwareEdit.ApplyWithOriginal(current, data)
		} else {
			updated, result, err = edit.Apply(current)
		}
		if err != nil {
			logger.Error("edit_failed", map[string]any{
				"index":       i,
				"type":        editType,
				"duration_ms": time.Since(start).Milliseconds(),
				"error":       err.Error(),
			})
			return nil, results, err
		}

		logger.Debug("edit_completed", map[string]any{
			"index":       i,
			"type":        editType,
			"changed":     result.Changed,
			"occurrences": result.Occurrences,
			"duration_ms": time.Since(start).Milliseconds(),
		})

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
