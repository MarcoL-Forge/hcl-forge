package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// validChangeTypes is the complete set of supported operation types.
var validChangeTypes = map[string]bool{
	"set_attr":     true,
	"remove_attr":  true,
	"remove_block": true,
	"copy_block":   true,
	"add_block":    true,
}

// Load reads a transform.yaml file from path and returns a validated Spec.
// Default values are applied for any omitted optional fields.
func Load(path string) (*Spec, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading spec file %q: %w", path, err)
	}

	var spec Spec
	if err := yaml.Unmarshal(data, &spec); err != nil {
		return nil, fmt.Errorf("parsing spec file %q: %w", path, err)
	}

	applyDefaults(&spec)

	return &spec, nil
}

// applyDefaults sets sensible default values for omitted optional fields.
func applyDefaults(s *Spec) {
	if s.SourceDir == "" {
		s.SourceDir = "./templates"
	}

	if s.TargetDir == "" {
		s.TargetDir = "./output"
	}

	if s.Vars == nil {
		s.Vars = make(map[string]string)
	}

	if s.Flags == nil {
		s.Flags = make(map[string]bool)
	}
}

// Validate checks the spec for structural errors without touching the filesystem.
// Returns the first error encountered.
func (s *Spec) Validate() error {
	if len(s.Files) == 0 {
		return fmt.Errorf("spec must contain at least one file")
	}

	for i, f := range s.Files {
		if f.Path == "" {
			return fmt.Errorf("files[%d]: path is required", i)
		}

		for j, c := range f.Changes {
			if err := validateChange(c, i, j); err != nil {
				return err
			}
		}
	}

	return nil
}

// validateChange checks a single Change for required fields based on its type.
func validateChange(c Change, fileIdx, changeIdx int) error {
	loc := fmt.Sprintf("files[%d].changes[%d]", fileIdx, changeIdx)

	if !validChangeTypes[c.Type] {
		return fmt.Errorf(
			"%s: unknown type %q — valid types: set_attr, remove_attr, remove_block, copy_block, add_block",
			loc, c.Type,
		)
	}

	switch c.Type {
	case "set_attr":
		if c.Block == "" {
			return fmt.Errorf("%s (set_attr): block is required", loc)
		}

		if c.Attr == "" {
			return fmt.Errorf("%s (set_attr): attr is required", loc)
		}

		if c.Value == "" {
			return fmt.Errorf("%s (set_attr): value is required", loc)
		}

	case "remove_attr":
		if c.Block == "" {
			return fmt.Errorf("%s (remove_attr): block is required", loc)
		}

		if c.Attr == "" {
			return fmt.Errorf("%s (remove_attr): attr is required", loc)
		}

	case "remove_block":
		if c.Block == "" {
			return fmt.Errorf("%s (remove_block): block is required", loc)
		}

	case "copy_block":
		if c.Block == "" {
			return fmt.Errorf("%s (copy_block): block is required", loc)
		}

	case "add_block":
		if c.FromFile == "" {
			return fmt.Errorf("%s (add_block): from_file is required", loc)
		}
	}

	return nil
}

// ApplyOverrides merges CLI --var and --flag values into the spec, overriding
// any values defined in the YAML file.
//
// Format for each entry is "key=value". Flag values must be true/false/1/0/yes/no.
func (s *Spec) ApplyOverrides(vars, flags []string) error {
	for _, v := range vars {
		k, val, err := splitKeyValue(v)
		if err != nil {
			return fmt.Errorf("--var %q: %w", v, err)
		}

		s.Vars[k] = val
	}

	for _, f := range flags {
		k, val, err := splitKeyValue(f)
		if err != nil {
			return fmt.Errorf("--flag %q: %w", f, err)
		}

		bval, err := parseBool(val)
		if err != nil {
			return fmt.Errorf("--flag %q: %w", f, err)
		}

		s.Flags[k] = bval
	}

	return nil
}

// ParseBlockRef parses a dot-notation block identifier into its type and labels.
//
// Examples:
//
//	"resource.aws_instance.web" -> BlockRef{Type: "resource", Labels: ["aws_instance", "web"]}
//	"provider.aws"              -> BlockRef{Type: "provider",  Labels: ["aws"]}
//	"locals"                    -> BlockRef{Type: "locals",    Labels: []}
func ParseBlockRef(ref string) (BlockRef, error) {
	if ref == "" {
		return BlockRef{}, fmt.Errorf("block ref must not be empty")
	}

	parts := strings.SplitN(ref, ".", 2)
	blockType := parts[0]

	if len(parts) == 1 {
		return BlockRef{Type: blockType, Labels: []string{}}, nil
	}

	return BlockRef{Type: blockType, Labels: strings.Split(parts[1], ".")}, nil
}

// splitKeyValue splits a "key=value" string into its components.
func splitKeyValue(s string) (string, string, error) {
	idx := strings.Index(s, "=")
	if idx < 0 {
		return "", "", fmt.Errorf("expected key=value format, got %q", s)
	}

	return s[:idx], s[idx+1:], nil
}

// parseBool converts common boolean string representations to a bool.
func parseBool(s string) (bool, error) {
	switch strings.ToLower(s) {
	case "true", "1", "yes":
		return true, nil
	case "false", "0", "no":
		return false, nil
	default:
		return false, fmt.Errorf("value must be true/false, got %q", s)
	}
}