package document

import (
	"fmt"
	"os"
	"path/filepath"
)

func LoadFile(path string) ([]byte, error) {
	absPath, err := ResolvePath(path)
	if err != nil {
		return nil, fmt.Errorf("resolve path: %w", err)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("read file %q: %w", absPath, err)
	}

	return data, nil
}

func LoadFileWithPath(path string) ([]byte, string, error) {
	absPath, err := ResolvePath(path)
	if err != nil {
		return nil, "", fmt.Errorf("resolve path: %w", err)
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, "", fmt.Errorf("read file %q: %w", absPath, err)
	}

	return data, absPath, nil
}

func ResolvePath(path string) (string, error) {
	return resolvePath("", path)
}

func ResolvePathFrom(root, path string) (string, error) {
	return resolvePath(root, path)
}

func resolvePath(root, path string) (string, error) {
	if filepath.IsAbs(path) {
		return filepath.Clean(path), nil
	}

	base := root
	if base == "" {
		var err error
		base, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("get working directory: %w", err)
		}
	}

	return filepath.Abs(filepath.Join(base, path))
}