package config

import (
	"fmt"
	"os"
	"regexp"

	"gopkg.in/yaml.v3"
)

var envVarPattern = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)(:-([^}]*))?\}`)

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", path, err)
	}

	var root yaml.Node
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("parse yaml config %q: %w", path, err)
	}

	if err := expandEnvVarsInNode(&root); err != nil {
		return nil, fmt.Errorf("expand config variables %q: %w", path, err)
	}

	var cfg Config
	if err := root.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("parse yaml config %q: %w", path, err)
	}

	if err := Validate(cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func expandEnvVarsInNode(node *yaml.Node) error {
	if node == nil {
		return nil
	}

	if node.Kind == yaml.ScalarNode {
		expanded, err := expandEnvVars(node.Value)
		if err != nil {
			return err
		}
		node.Value = expanded
	}

	for i := range node.Content {
		if err := expandEnvVarsInNode(node.Content[i]); err != nil {
			return err
		}
	}

	return nil
}

func expandEnvVars(in string) (string, error) {
	var firstErr error

	out := envVarPattern.ReplaceAllStringFunc(in, func(match string) string {
		if firstErr != nil {
			return match
		}

		parts := envVarPattern.FindStringSubmatch(match)
		if len(parts) < 2 {
			firstErr = fmt.Errorf("invalid variable expression %q", match)
			return match
		}

		name := parts[1]
		if value, ok := os.LookupEnv(name); ok {
			return value
		}

		if len(parts) >= 4 && parts[3] != "" {
			return parts[3]
		}

		firstErr = fmt.Errorf("missing required environment variable %q", name)
		return match
	})

	if firstErr != nil {
		return "", firstErr
	}

	return out, nil
}
