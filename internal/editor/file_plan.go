package editor

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Marc0l95/hclforge/internal/document"
	"github.com/Marc0l95/hclforge/internal/logging"
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
	Error      string
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
	return runFilePlans(plans, clampWorkers(workers, 5), PlanFilePlan)
}

func ApplyFilePlans(plans []FilePlan, workers int) ([]FilePlanResult, error) {
	return runFilePlans(plans, clampWorkers(workers, 10), ApplyFilePlan)
}

func runFilePlans(
	plans []FilePlan,
	workers int,
	fn func(FilePlan) (FilePlanResult, error),
) ([]FilePlanResult, error) {
	if err := validateUniqueOutputPaths(plans); err != nil {
		return nil, err
	}

	jobs := make(chan FilePlanJob)
	results := make([]FilePlanResult, len(plans))

	var wg sync.WaitGroup

	var errMu sync.Mutex
	errs := make([]error, 0)
	logger := logging.Default()

	for workerID := 0; workerID < workers; workerID++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for job := range jobs {
				start := time.Now()
				logger.Debug("file_job_start", map[string]any{
					"index":  job.Index,
					"source": job.Plan.SourcePath,
					"output": job.Plan.OutputPath,
				})

				result := FilePlanResult{
					SourcePath: job.Plan.SourcePath,
					OutputPath: job.Plan.OutputPath,
				}

				result, err := fn(job.Plan)
				if err != nil {
					result.Error = err.Error()
					results[job.Index] = result
					logger.Error("file_job_failed", map[string]any{
						"index":       job.Index,
						"source":      job.Plan.SourcePath,
						"output":      job.Plan.OutputPath,
						"error":       err.Error(),
						"duration_ms": time.Since(start).Milliseconds(),
					})

					errMu.Lock()
					errs = append(errs, err)
					errMu.Unlock()
					continue
				}

				results[job.Index] = result
				logger.Debug("file_job_completed", map[string]any{
					"index":       job.Index,
					"source":      result.SourcePath,
					"output":      result.OutputPath,
					"changed":     result.Changed,
					"duration_ms": time.Since(start).Milliseconds(),
				})
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

	if len(errs) > 0 {
		return results, fmt.Errorf("%d file plan(s) failed: %w", len(errs), errors.Join(errs...))
	}

	return results, nil
}

func clampWorkers(workers, max int) int {
	if workers <= 0 {
		workers = 4
	}

	if workers > max {
		workers = max
	}

	return workers
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
