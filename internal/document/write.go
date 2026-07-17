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

	tmpFile, err := os.CreateTemp(filepath.Dir(absPath), filepath.Base(absPath)+".tmp-*")
	if err != nil {
		return fmt.Errorf("create temp file for %q: %w", absPath, err)
	}
	tmpPath := tmpFile.Name()

	defer func() {
		_ = os.Remove(tmpPath)
	}()

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("write temp file %q: %w", tmpPath, err)
	}

	if err := tmpFile.Sync(); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("sync temp file %q: %w", tmpPath, err)
	}

	if err := tmpFile.Chmod(0o644); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("chmod temp file %q: %w", tmpPath, err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp file %q: %w", tmpPath, err)
	}

	if err := os.Rename(tmpPath, absPath); err != nil {
		return fmt.Errorf("replace file %q atomically: %w", absPath, err)
	}

	if err := syncDir(filepath.Dir(absPath)); err != nil {
		return fmt.Errorf("sync directory %q after rename: %w", filepath.Dir(absPath), err)
	}

	return nil
}

func syncDir(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() {
		_ = dir.Close()
	}()

	return dir.Sync()
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
