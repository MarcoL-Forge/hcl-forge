package editor

import (
	"fmt"

	"github.com/Marc0l95/hclforge/internal/document"
)

type FilePlan struct {
	Path  string
	Edits []Edit
}

type FilePlanResult struct {
	Path    string
	Results []EditResult
	Changed bool
}

func ApplyFilePlan(plan FilePlan) (FilePlanResult, error) {
	data, absPath, err := document.LoadFileWithPath(plan.Path)
	if err != nil {
		return FilePlanResult{}, err
	}

	updated, results, err := ApplyEdits(data, plan.Edits)
	if err != nil {
		return FilePlanResult{}, fmt.Errorf("apply edits to %q: %w", absPath, err)
	}

	changed := HasChanges(results)
	if changed {
		if err := document.WriteFile(absPath, updated); err != nil {
			return FilePlanResult{}, fmt.Errorf("write updated file %q: %w", absPath, err)
		}
	}

	return FilePlanResult{
		Path:    absPath,
		Results: results,
		Changed: changed,
	}, nil
}

func ApplyFilePlans(plans []FilePlan) ([]FilePlanResult, error) {
	results := make([]FilePlanResult, 0, len(plans))

	for _, plan := range plans {
		result, err := ApplyFilePlan(plan)
		if err != nil {
			return results, err
		}

		results = append(results, result)
	}

	return results, nil
}