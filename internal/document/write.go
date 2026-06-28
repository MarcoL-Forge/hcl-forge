package document

import (
	"fmt"
	"os"
	"path/filepath"
)

func WriteFile(path string, data []byte) error {
	absPath, err := ResolvePath(path)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return fmt.Errorf("create directories for %q: %w", absPath, err)
	}

	if err := os.WriteFile(absPath, data, 0o644); err != nil {
		return fmt.Errorf("write file %q: %w", absPath, err)
	}

	return nil
}

func WriteToTargetDir(sourcePath, targetDir string, data []byte) (string, error) {
	absTargetDir, err := ResolvePath(targetDir)
	if err != nil {
		return "", fmt.Errorf("resolve target directory: %w", err)
	}

	fileName := filepath.Base(sourcePath)
	if fileName == "." || fileName == string(filepath.Separator) {
		fileName = "output.tf"
	}

	outPath := filepath.Join(absTargetDir, fileName)

	if err := WriteFile(outPath, data); err != nil {
		return "", err
	}

	return outPath, nil
}
