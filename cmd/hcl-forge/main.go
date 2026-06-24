package main

import (
	"fmt"
	"os"

	"github.com/Marc0l95/hclforge/internal/document"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("usage: hcl-forge <source file> <target directory>")
		os.Exit(1)
	}

	inputDir := os.Args[1]
	outputDir := os.Args[2]

	raw, err := document.LoadFile(inputDir)
	if err != nil {
		fmt.Printf("load file: %v\n", err)
		os.Exit(1)
	}

	editedRaw := raw

	outputPath, err := document.WriteToTargetDir(inputDir, outputDir, editedRaw)
	if err != nil {
		fmt.Printf("write file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("File written to: %s\n", outputPath)
}
