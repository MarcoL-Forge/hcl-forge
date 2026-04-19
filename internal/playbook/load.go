package playbook

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

func Load(path string) (*Playbook, error) {
	if path == "" {
		return nil, fmt.Errorf("playbook path is empty")
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read playbook: %w", err)
	}

	var pb Playbook
	if err := yaml.Unmarshal(raw, &pb); err != nil {
		return nil, fmt.Errorf("parse playbook yaml: %w", err)
	}

	if pb.Version == 0 {
		pb.Version = 1
	}

	if pb.Input == "" {
		return nil, fmt.Errorf("playbook input is empty")
	}

	if len(pb.Operations) == 0 {
		return nil, fmt.Errorf("playbook operations are empty")
	}

	baseDir := filepath.Dir(path)
	pb.Input = resolvePath(baseDir, pb.Input)
	if pb.Output != "" {
		pb.Output = resolvePath(baseDir, pb.Output)
	}

	return &pb, nil
}

func resolvePath(baseDir string, path string) string {
	if path == "" || filepath.IsAbs(path) {
		return path
	}

	return filepath.Join(baseDir, path)
}
