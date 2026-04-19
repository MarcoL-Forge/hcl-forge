package cli

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/Marc0l95/hcl-forge/internal/document"
	"github.com/Marc0l95/hcl-forge/internal/parser"
)

func runReplace(args []string) error {
	replaceFlags := flag.NewFlagSet("replace", flag.ContinueOnError)
	replaceFlags.SetOutput(os.Stderr)

	inputPath := replaceFlags.String("in", "", "input Terraform or HCL file path")
	outputPath := replaceFlags.String("out", "", "output file path")
	writeStdout := replaceFlags.Bool("stdout", false, "print replaced content to stdout")
	blockType := replaceFlags.String("block-type", "", "target block type, for example resource or module")
	labels := replaceFlags.String("labels", "", "comma-separated block labels")
	attribute := replaceFlags.String("attr", "", "target attribute name")
	value := replaceFlags.String("value", "", "replacement value")
	valueType := replaceFlags.String("value-type", "string", "replacement value type: string, bool, or number")

	if err := replaceFlags.Parse(args); err != nil {
		return err
	}

	if *inputPath == "" {
		return fmt.Errorf("replace requires --in")
	}

	if *blockType == "" {
		return fmt.Errorf("replace requires --block-type")
	}

	if *attribute == "" {
		return fmt.Errorf("replace requires --attr")
	}

	if *outputPath == "" && !*writeStdout {
		return fmt.Errorf("replace requires at least one output target: --out or --stdout")
	}

	doc, err := document.LoadDocument(*inputPath)
	if err != nil {
		return fmt.Errorf("load document: %w", err)
	}

	replacedDoc, err := parser.ReplaceAttributeValue(doc, parser.ReplaceAttributeInput{
		BlockType: *blockType,
		Labels:    parseLabels(*labels),
		Attribute: *attribute,
		Value:     *value,
		ValueType: *valueType,
	})
	if err != nil {
		return fmt.Errorf("replace attribute: %w", err)
	}

	renderedRaw, err := document.RenderDocument(replacedDoc)
	if err != nil {
		return fmt.Errorf("render document: %w", err)
	}

	if *writeStdout {
		fmt.Print(string(renderedRaw))
	}

	if *outputPath != "" {
		if err := document.WriteDocument(&document.Document{Path: replacedDoc.Path, Raw: renderedRaw}, *outputPath); err != nil {
			return fmt.Errorf("write output file: %w", err)
		}

		fmt.Fprintf(os.Stderr, "wrote replaced file to %s\n", *outputPath)
	}

	return nil
}

func parseLabels(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	labels := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			labels = append(labels, trimmed)
		}
	}

	return labels
}
