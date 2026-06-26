package cli

import (
	"fmt"

	"github.com/Marc0l95/hclforge/internal/editor"
)

func printResults(mode string, results []editor.FilePlanResult) {
	fmt.Printf("%s results:\n\n", mode)

	for _, fileResult := range results {
		fmt.Printf("Source:  %s\n", fileResult.SourcePath)
		fmt.Printf("Output:  %s\n", fileResult.OutputPath)
		fmt.Printf("Changed: %v\n", fileResult.Changed)

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