package config

import "fmt"

func (cfg *Config) ApplyDefaults() {
	if cfg.Input.RootDir == "" {
		cfg.Input.RootDir = "."
	}

	if cfg.Options.Workers <= 0 {
		cfg.Options.Workers = 4
	}

	if cfg.Output.Mode == "" {
		cfg.Output.Mode = "overwrite"
	}
}

func Validate(cfg Config) error {
	if cfg.Version != 1 {
		return fmt.Errorf("unsupported config version %d", cfg.Version)
	}

	if len(cfg.Input.Files) == 0 {
		return fmt.Errorf("input.files must contain at least one file")
	}

	switch cfg.Output.Mode {
	case "overwrite", "target_dir":
	default:
		return fmt.Errorf("unsupported output mode %q", cfg.Output.Mode)
	}

	if cfg.Output.Mode == "target_dir" && cfg.Output.TargetDir == "" {
		return fmt.Errorf("output.target_dir is required when output.mode is target_dir")
	}

	if len(cfg.Edits) == 0 {
		return fmt.Errorf("config must contain at least one edit")
	}

	for i, edit := range cfg.Edits {
		if edit.Type == "" {
			return fmt.Errorf("edits[%d]: missing type", i)
		}

		switch edit.Type {
		case "search_replace":
			if edit.Old == "" {
				return fmt.Errorf("edits[%d]: search_replace requires old", i)
			}

		case "insert_hcl":
			if edit.HCL == "" {
				return fmt.Errorf("edits[%d]: insert_hcl requires hcl", i)
			}
			if edit.Block != nil && edit.Block.SelectedType() == "" {
				return fmt.Errorf("edits[%d]: block selector requires block_type (or type)", i)
			}

		case "delete_hcl":
			if edit.Block != nil && edit.Block.SelectedType() == "" {
				return fmt.Errorf("edits[%d]: block selector requires block_type (or type)", i)
			}
			if edit.Attribute == "" && edit.Block == nil {
				return fmt.Errorf("edits[%d]: delete_hcl requires attribute or block", i)
			}

		default:
			return fmt.Errorf("edits[%d]: unsupported edit type %q", i, edit.Type)
		}
	}

	return nil
}
