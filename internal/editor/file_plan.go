package editor

import (
	"fmt"
	"sync"

	"github.com/Marc0l95/hclforge/internal/document"
)

type FilePlan struct {
	SourcePath string
	OutputPath string
	Edits      []Edit
}

type FilePlanResult struct {
	SourcePath string
	OutputPath string
	Results    []EditResult
	Changed    bool
}

type FilePlanJob struct {
	Index int
	Plan  FilePlan
}

func PlanFilePlan(plan FilePlan) (FilePlanResult, error) {
	data, absSourcePath, err := document.LoadFileWithPath(plan.SourcePath)
	if err != nil {
		return FilePlanResult{}, err
	}

	_, results, err := ApplyEdits(data, plan.Edits)
	if err != nil {
		return FilePlanResult{}, fmt.Errorf("plan edits for %q: %w", absSourcePath, err)
	}

	return FilePlanResult{
		SourcePath: absSourcePath,
		OutputPath: plan.OutputPath,
		Results:    results,
		Changed:    HasChanges(results),
	}, nil
}

func ApplyFilePlan(plan FilePlan) (FilePlanResult, error) {
	data, absSourcePath, err := document.LoadFileWithPath(plan.SourcePath)
	if err != nil {
		return FilePlanResult{}, err
	}

	updated, results, err := ApplyEdits(data, plan.Edits)
	if err != nil {
		return FilePlanResult{}, fmt.Errorf("apply edits to %q: %w", absSourcePath, err)
	}

	changed := HasChanges(results)
	if changed {
		if err := document.WriteFile(plan.OutputPath, updated); err != nil {
			return FilePlanResult{}, fmt.Errorf("write updated file %q: %w", plan.OutputPath, err)
		}
	}

	return FilePlanResult{
		SourcePath: absSourcePath,
		OutputPath: plan.OutputPath,
		Results:    results,
		Changed:    changed,
	}, nil
}

func PlanFilePlans(plans []FilePlan, workers int) ([]FilePlanResult, error) {
	return runFilePlans(plans, workers, PlanFilePlan)
}

func ApplyFilePlans(plans []FilePlan, workers int) ([]FilePlanResult, error) {
	return runFilePlans(plans, workers, ApplyFilePlan)
}

func runFilePlans(
	plans []FilePlan,
	workers int,
	fn func(FilePlan) (FilePlanResult, error),
) ([]FilePlanResult, error) {
	if workers <= 0 {
		workers = 4
	}

	if err := validateUniqueOutputPaths(plans); err != nil {
		return nil, err
	}

	jobs := make(chan FilePlanJob)
	results := make([]FilePlanResult, len(plans))

	var wg sync.WaitGroup

	var firstErr error
	var errMu sync.Mutex

	for workerID := 0; workerID < workers; workerID++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for job := range jobs {
				result, err := fn(job.Plan)
				if err != nil {
					errMu.Lock()
					if firstErr == nil {
						firstErr = err
					}
					errMu.Unlock()
					continue
				}

				results[job.Index] = result
			}
		}()
	}

	for index, plan := range plans {
		jobs <- FilePlanJob{
			Index: index,
			Plan:  plan,
		}
	}

	close(jobs)
	wg.Wait()

	if firstErr != nil {
		return results, firstErr
	}

	return results, nil
}

func validateUniqueOutputPaths(plans []FilePlan) error {
	seen := make(map[string]string, len(plans))

	for _, plan := range plans {
		absOutputPath, err := document.ResolvePath(plan.OutputPath)
		if err != nil {
			return fmt.Errorf("resolve output path %q: %w", plan.OutputPath, err)
		}

		if existingSource, exists := seen[absOutputPath]; exists {
			return fmt.Errorf(
				"duplicate output path %q for source files %q and %q",
				absOutputPath,
				existingSource,
				plan.SourcePath,
			)
		}

		seen[absOutputPath] = plan.SourcePath
	}

	return nil
}
