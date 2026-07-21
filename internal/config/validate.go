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

	if len(cfg.Output.FileMap) > 0 {
		if cfg.Output.Mode != "target_dir" {
			return fmt.Errorf("output.file_map is only supported when output.mode is target_dir")
		}

		inputFiles := make(map[string]struct{}, len(cfg.Input.Files))
		for _, file := range cfg.Input.Files {
			inputFiles[file] = struct{}{}
		}

		for inputFile, outputFile := range cfg.Output.FileMap {
			if inputFile == "" {
				return fmt.Errorf("output.file_map cannot contain an empty input file key")
			}

			if outputFile == "" {
				return fmt.Errorf("output.file_map[%q] cannot be empty", inputFile)
			}

			if _, ok := inputFiles[inputFile]; !ok {
				return fmt.Errorf("output.file_map[%q] does not match any input.files entry", inputFile)
			}
		}
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

			switch edit.MatchMode {
			case "", "glob", "regex":
			default:
				return fmt.Errorf("edits[%d]: search_replace match_mode must be one of glob|regex", i)
			}

			if edit.Block != nil {
				if edit.Attribute == "" {
					return fmt.Errorf("edits[%d]: search_replace with block selector requires attribute", i)
				}

				if err := validateBlockSelector(*edit.Block); err != nil {
					return fmt.Errorf("edits[%d]: %w", i, err)
				}
			}

		case "insert_hcl":
			if edit.HCL == "" {
				return fmt.Errorf("edits[%d]: insert_hcl requires hcl", i)
			}

			if edit.Placement != nil {
				mode := edit.Placement.Mode
				switch mode {
				case "append", "prepend", "after_attribute", "before_attribute":
				default:
					return fmt.Errorf("edits[%d]: insert_hcl placement.mode must be one of append|prepend|after_attribute|before_attribute", i)
				}

				switch mode {
				case "after_attribute", "before_attribute":
					if edit.Placement.Attribute == "" {
						return fmt.Errorf("edits[%d]: insert_hcl placement.attribute is required for mode %q", i, mode)
					}
				default:
					if edit.Placement.Attribute != "" {
						return fmt.Errorf("edits[%d]: insert_hcl placement.attribute is only supported for before_attribute|after_attribute", i)
					}
				}
			}

			if edit.Guard != nil && edit.Guard.IfTargetExists && edit.Guard.IfTargetMissing {
				return fmt.Errorf("edits[%d]: guard.if_target_exists and guard.if_target_missing cannot both be true", i)
			}

			if (edit.EnsureTargetBlock || edit.Guard != nil) && edit.Block == nil {
				return fmt.Errorf("edits[%d]: insert_hcl ensure/guard requires block selector", i)
			}

			if edit.Block != nil {
				if err := validateBlockSelector(*edit.Block); err != nil {
					return fmt.Errorf("edits[%d]: %w", i, err)
				}
			}

		case "delete_hcl":
			switch edit.MatchMode {
			case "", "glob", "regex":
			default:
				return fmt.Errorf("edits[%d]: delete_hcl match_mode must be one of glob|regex", i)
			}

			if edit.KeepOnly && edit.Block == nil {
				return fmt.Errorf("edits[%d]: delete_hcl keep_only requires block selector", i)
			}

			if edit.KeepOnly && edit.Attribute != "" {
				return fmt.Errorf("edits[%d]: delete_hcl keep_only cannot be combined with attribute", i)
			}

			if edit.Block != nil {
				if err := validateBlockSelector(*edit.Block); err != nil {
					return fmt.Errorf("edits[%d]: %w", i, err)
				}
			}
			if edit.Attribute == "" && edit.Block == nil {
				return fmt.Errorf("edits[%d]: delete_hcl requires attribute or block", i)
			}

		case "set_attribute":
			if edit.Attribute == "" {
				return fmt.Errorf("edits[%d]: set_attribute requires attribute", i)
			}

			if edit.ValueHCL == "" {
				return fmt.Errorf("edits[%d]: set_attribute requires value_hcl", i)
			}

			if edit.Block != nil {
				if err := validateBlockSelector(*edit.Block); err != nil {
					return fmt.Errorf("edits[%d]: %w", i, err)
				}
			}

		default:
			return fmt.Errorf("edits[%d]: unsupported edit type %q", i, edit.Type)
		}
	}

	return nil
}

func validateBlockSelector(block BlockSelector) error {
	if block.Path != "" {
		if block.SelectedType() != "" || len(block.Labels) > 0 || len(block.Parents) > 0 {
			return fmt.Errorf("block.path cannot be combined with block_type/type, labels, or parents")
		}

		if _, err := selectorFromPath(block.Path); err != nil {
			return fmt.Errorf("invalid block.path: %w", err)
		}

		return nil
	}

	if block.SelectedType() == "" {
		return fmt.Errorf("block selector requires block_type (or type)")
	}

	for j, parent := range block.Parents {
		if parent.SelectedType() == "" {
			return fmt.Errorf("block.parents[%d] requires block_type (or type)", j)
		}
	}

	return nil
}
