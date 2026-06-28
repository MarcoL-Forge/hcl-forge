package config

import (
	"fmt"

	"github.com/Marc0l95/hclforge/internal/document"
	"github.com/Marc0l95/hclforge/internal/editor"
)

func BuildFilePlans(cfg Config) ([]editor.FilePlan, error) {
	edits := make([]editor.Edit, 0, len(cfg.Edits))

	for i, editCfg := range cfg.Edits {
		edit, err := buildEdit(editCfg)
		if err != nil {
			return nil, fmt.Errorf("edits[%d]: %w", i, err)
		}

		edits = append(edits, edit)
	}

	plans := make([]editor.FilePlan, 0, len(cfg.Input.Files))

	for _, file := range cfg.Input.Files {
		sourcePath, err := document.ResolvePathFrom(cfg.Input.RootDir, file)
		if err != nil {
			return nil, fmt.Errorf("resolve source path %q: %w", file, err)
		}

		var outputPath string

		switch cfg.Output.Mode {
		case "overwrite":
			outputPath = sourcePath

		case "target_dir":
			outputPath, err = document.ResolvePathFrom(cfg.Output.TargetDir, file)
			if err != nil {
				return nil, fmt.Errorf("resolve output path %q: %w", file, err)
			}

		default:
			return nil, fmt.Errorf("unsupported output mode %q", cfg.Output.Mode)
		}

		plans = append(plans, editor.FilePlan{
			SourcePath: sourcePath,
			OutputPath: outputPath,
			Edits:      edits,
		})
	}

	return plans, nil
}

func buildEdit(editCfg EditConfig) (editor.Edit, error) {
	switch editCfg.Type {
	case "search_replace":
		return editor.SearchReplaceEdit{
			Old: editCfg.Old,
			New: editCfg.New,
		}, nil

	case "insert_hcl":
		var targetBlock *editor.BlockSelector
		if editCfg.Block != nil {
			resolved, err := resolveBlockSelector(*editCfg.Block)
			if err != nil {
				return nil, err
			}

			parents := make([]editor.ParentSelector, 0, len(resolved.Parents))
			for _, parent := range resolved.Parents {
				parents = append(parents, editor.ParentSelector{
					Type:   parent.SelectedType(),
					Labels: append([]string(nil), parent.Labels...),
				})
			}

			targetBlock = &editor.BlockSelector{
				Type:    resolved.Type,
				Labels:  append([]string(nil), resolved.Labels...),
				Parents: parents,
			}
		}

		return editor.InsertHCLEdit{
			HCL:               editCfg.HCL,
			TargetBlock:       targetBlock,
			EnsureTargetBlock: editCfg.EnsureTargetBlock,
			Guard: &editor.InsertGuard{
				IfTargetExists:  editCfg.Guard != nil && editCfg.Guard.IfTargetExists,
				IfTargetMissing: editCfg.Guard != nil && editCfg.Guard.IfTargetMissing,
			},
		}, nil

	case "delete_hcl":
		var targetBlock *editor.BlockSelector
		if editCfg.Block != nil {
			resolved, err := resolveBlockSelector(*editCfg.Block)
			if err != nil {
				return nil, err
			}

			parents := make([]editor.ParentSelector, 0, len(resolved.Parents))
			for _, parent := range resolved.Parents {
				parents = append(parents, editor.ParentSelector{
					Type:   parent.SelectedType(),
					Labels: append([]string(nil), parent.Labels...),
				})
			}

			targetBlock = &editor.BlockSelector{
				Type:    resolved.Type,
				Labels:  append([]string(nil), resolved.Labels...),
				Parents: parents,
			}
		}

		return editor.DeleteHCLEdit{
			TargetBlock: targetBlock,
			Attribute:   editCfg.Attribute,
			DeleteAll:   editCfg.DeleteAll,
		}, nil

	default:
		return nil, fmt.Errorf("unsupported edit type %q", editCfg.Type)
	}
}

func resolveBlockSelector(block BlockSelector) (resolvedSelector, error) {
	if block.Path != "" {
		return selectorFromPath(block.Path)
	}

	return resolvedSelector{
		Type:    block.SelectedType(),
		Labels:  append([]string(nil), block.Labels...),
		Parents: append([]ParentSelector(nil), block.Parents...),
	}, nil
}
