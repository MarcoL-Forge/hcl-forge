package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Marc0l95/hcl-forge/internal/document"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: hcl-forge <input-file>")
		os.Exit(1)
	}

	inputPath := os.Args[1]
	doc, err := document.LoadDocument(inputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load document: %v\n", err)
		os.Exit(1)
	}

	moduleRoot, err := findModuleRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "find module root: %v\n", err)
		os.Exit(1)
	}

	renderedRaw, err := document.RenderDocument(doc)
	if err != nil {
		fmt.Fprintf(os.Stderr, "render document: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("rendered %s:\n\n%s\n", doc.Path, renderedRaw)

	outputPath := filepath.Join(moduleRoot, "output.tf")
	if err := document.WriteDocument(doc, outputPath); err != nil {
		fmt.Fprintf(os.Stderr, "write output file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("loaded %s and wrote %s\n", doc.Path, outputPath)
}

func findModuleRoot() (string, error) {
	workingDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	currentDir := workingDir
	for {
		if _, err := os.Stat(filepath.Join(currentDir, "go.mod")); err == nil {
			return currentDir, nil
		}

		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			return "", fmt.Errorf("go.mod not found from %s upward", workingDir)
		}

		currentDir = parentDir
	}
}
