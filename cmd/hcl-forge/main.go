package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/Marc0l95/hclforge/internal/editor"
	"github.com/Marc0l95/hclforge/internal/document"
)

func main() {
	inputPath := flag.String("in", "", "input .tf file path")
	outputDir := flag.String("out", "", "optional output directory; if empty, overwrites input file")
	oldValue := flag.String("old", "", "text to search for")
	newValue := flag.String("new", "", "replacement text")

	flag.Parse()

	if *inputPath == "" {
		log.Fatal("missing required flag: -in")
	}

	if *oldValue == "" {
		log.Fatal("missing required flag: -old")
	}

	data, absPath, err := document.LoadFileWithPath(*inputPath)
	if err != nil {
		log.Fatal(err)
	}

	updated, results, err := editor.ApplyEdits(data, []editor.Edit{
		editor.SearchReplaceEdit{
			Old: *oldValue,
			New: *newValue,
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	changed := editor.HasChanges(results)

	if !changed {
		fmt.Printf("No changes made to %s\n", absPath)
		return
	}

	var writtenPath string

	if *outputDir != "" {
		writtenPath, err = document.WriteToTargetDir(absPath, *outputDir, updated)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		if err := document.WriteFile(absPath, updated); err != nil {
			log.Fatal(err)
		}
		writtenPath = absPath
	}

	fmt.Printf("Updated file: %s\n", writtenPath)

	for _, result := range results {
		fmt.Printf(
			"- %s, occurrences=%d\n",
			result.Message,
			result.Occurrences,
		)
	}
}
