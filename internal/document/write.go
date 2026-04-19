package document

import (
	"fmt"
	"os"
	"path/filepath"
)

func WriteDocument(doc *Document, outputPath string) error {
	if doc == nil {
		return fmt.Errorf("document is nil")
	}

	if outputPath == "" {
		return fmt.Errorf("output path is empty")
	}

	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}

	if err := os.WriteFile(outputPath, doc.Raw, 0o600); err != nil {
		return fmt.Errorf("write document: %w", err)
	}

	return nil
}
