package document

import (
	"fmt"
	"io"
	"os"
)

func LoadDocument(path string) (*Document, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	raw := make([]byte, info.Size())
	_, err = io.ReadFull(file, raw)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	return &Document{
		Path: path,
		Raw:  raw,
	}, nil
}
