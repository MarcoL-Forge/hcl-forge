package cli

import (
	"fmt"

	"github.com/MarcoL-Forge/hcl-forge/internal/editor"
	"github.com/MarcoL-Forge/hcl-forge/internal/logging"
)

func printResults(mode string, results []editor.FilePlanResult, logger *logging.Logger, quiet bool) {
	if !quiet {
		fmt.Printf("%s results:\n\n", mode)
	}

	changedCount := 0
	failedCount := 0

	for _, fileResult := range results {
		if !quiet {
			fmt.Printf("Source:  %s\n", fileResult.SourcePath)
			fmt.Printf("Output:  %s\n", fileResult.OutputPath)
			fmt.Printf("Changed: %v\n", fileResult.Changed)
		}

		logFields := map[string]any{
			"mode":    mode,
			"source":  fileResult.SourcePath,
			"output":  fileResult.OutputPath,
			"changed": fileResult.Changed,
		}

		if fileResult.Changed {
			changedCount++
		}

		if fileResult.Error != "" {
			failedCount++
			if !quiet {
				fmt.Printf("  - error: %s\n", fileResult.Error)
			}
			logFields["error"] = fileResult.Error
			logger.Error("file_result", logFields)
		} else {
			logger.Info("file_result", logFields)
		}

		if !quiet {
			for _, editResult := range fileResult.Results {
				fmt.Printf(
					"  - %s, occurrences=%d, changed=%v\n",
					editResult.Message,
					editResult.Occurrences,
					editResult.Changed,
				)
			}

			fmt.Println()
		}
	}

	if !quiet {
		fmt.Printf("Summary: files=%d changed=%d failed=%d\n", len(results), changedCount, failedCount)
	}
	logger.Info("summary", map[string]any{
		"mode":    mode,
		"files":   len(results),
		"changed": changedCount,
		"failed":  failedCount,
	})
}
