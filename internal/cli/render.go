package cli

import (
	"flag"
	"fmt"
	"os"

	"github.com/Marc0l95/hcl-forge/internal/document"
)

func runRender(args []string) error {
	renderFlags := flag.NewFlagSet("render", flag.ContinueOnError)
	renderFlags.SetOutput(os.Stderr)

	inputPath := renderFlags.String("in", "", "input Terraform or HCL file path")
	outputPath := renderFlags.String("out", "", "output file path")
	writeStdout := renderFlags.Bool("stdout", false, "print rendered content to stdout")

	if err := renderFlags.Parse(args); err != nil {
		return err
	}

	if *inputPath == "" {
		return fmt.Errorf("render requires --in")
	}

	if *outputPath == "" && !*writeStdout {
		return fmt.Errorf("render requires at least one output target: --out or --stdout")
	}

	doc, err := document.LoadDocument(*inputPath)
	if err != nil {
		return fmt.Errorf("load document: %w", err)
	}

	renderedRaw, err := document.RenderDocument(doc)
	if err != nil {
		return fmt.Errorf("render document: %w", err)
	}

	if *writeStdout {
		fmt.Print(string(renderedRaw))
	}

	if *outputPath != "" {
		renderedDoc := &document.Document{
			Path: doc.Path,
			Raw:  renderedRaw,
		}

		if err := document.WriteDocument(renderedDoc, *outputPath); err != nil {
			return fmt.Errorf("write output file: %w", err)
		}

		fmt.Fprintf(os.Stderr, "wrote rendered file to %s\n", *outputPath)
	}

	return nil
}
