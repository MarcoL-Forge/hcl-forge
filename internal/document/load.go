package document

import (
	"fmt"
	"os"
	"path/filepath"
)

func LoadFile(path string) ([]byte, error) {
	absPath, err := resolvePath("", path)
	if err != nil {
		return nil, fmt.Errorf("resolve path: %w", err)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("read file %q: %w", absPath, err)
	}

	return data, nil
}

func resolvePath(root, p string) (string, error) {
	if filepath.IsAbs(p) {
		return filepath.Clean(p), nil
	}
	base := root
	if base == "" {
		// If no explicit root is provided, resolve relative paths from the
		// process working directory (for example local terminal cwd or CI job cwd).
		var err error
		base, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("get working directory: %w", err)
		}
	}
	return filepath.Abs(filepath.Join(base, p))
}
