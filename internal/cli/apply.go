package cli

import (
	"flag"
	"fmt"
	"os"

	"github.com/Marc0l95/hcl-forge/internal/document"
	"github.com/Marc0l95/hcl-forge/internal/parser"
	"github.com/Marc0l95/hcl-forge/internal/playbook"
)

func runApply(args []string) error {
	applyFlags := flag.NewFlagSet("apply", flag.ContinueOnError)
	applyFlags.SetOutput(os.Stderr)

	playbookPath := applyFlags.String("playbook", "", "path to a YAML playbook")
	outputOverride := applyFlags.String("out", "", "override output file path")
	writeStdout := applyFlags.Bool("stdout", false, "print rendered content to stdout")

	if err := applyFlags.Parse(args); err != nil {
		return err
	}

	if *playbookPath == "" {
		return fmt.Errorf("apply requires --playbook")
	}

	pb, err := playbook.Load(*playbookPath)
	if err != nil {
		return fmt.Errorf("load playbook: %w", err)
	}

	doc, err := document.LoadDocument(pb.Input)
	if err != nil {
		return fmt.Errorf("load document: %w", err)
	}

	for _, op := range pb.Operations {
		switch op.Op {
		case "set_attribute":
			doc, err = parser.ReplaceAttributeValue(doc, parser.ReplaceAttributeInput{
				Selector:  op.Target,
				BlockType: op.BlockType,
				Labels:    op.Labels,
				Attribute: op.Attribute,
				Value:     op.Value,
				ValueType: op.ValueType,
			})
			if err != nil {
				return fmt.Errorf("apply set_attribute: %w", err)
			}
		default:
			return fmt.Errorf("unsupported playbook operation %q", op.Op)
		}
	}

	renderedRaw, err := document.RenderDocument(doc)
	if err != nil {
		return fmt.Errorf("render document: %w", err)
	}

	if *writeStdout {
		fmt.Print(string(renderedRaw))
	}

	outputPath := pb.Output
	if *outputOverride != "" {
		outputPath = *outputOverride
	}

	if outputPath != "" {
		if err := document.WriteDocument(&document.Document{Path: doc.Path, Raw: renderedRaw}, outputPath); err != nil {
			return fmt.Errorf("write output file: %w", err)
		}

		fmt.Fprintf(os.Stderr, "wrote playbook output to %s\n", outputPath)
	}

	if outputPath == "" && !*writeStdout {
		return fmt.Errorf("apply requires an output target from the playbook, --out, or --stdout")
	}

	return nil
}
