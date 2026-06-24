# hcl-forge

`hcl-forge` is a Go-based CLI and library for safely manipulating Terraform/HCL files while preserving existing formatting and comments as much as possible.

## Intent

- Ingest one or many Terraform/HCL source files.
- Parse and index Terraform objects (blocks, attributes, selectors).
- Apply safe transformations (set/remove/add/update) using operations.
- Write output to one or many target paths (single, mirror, or mapped routing).
- Keep the workflow user-friendly through concise CLI commands and YAML-driven configuration.
- Provide a dry-run/plan mode with diffs before writing changes.

## Builder Checklist

Use this as the implementation checklist while building out the project.

### Foundation

- [ ] Initialize Go module and baseline project structure (`cmd`, `internal`, `pkg` if needed).
- [ ] Add parser/document round-trip support for `.tf`, `.tfvars`, and `.hcl`.
- [ ] Add deterministic write pipeline that minimizes source drift.

### Targeting + Operations

- [ ] Define selector grammar for block and attribute targeting.
- [ ] Implement `set_attribute`.
- [ ] Implement `remove_attribute`.
- [ ] Implement `remove_block`.
- [ ] Implement `add_block` via snippets/templates.
- [ ] Add validation for missing/ambiguous selectors.

### Multi-file Routing

- [ ] Accept single file, multiple files, and glob patterns as input.
- [ ] Add routing modes: single output, mirrored output tree, explicit mapping.
- [ ] Support one-to-many output fan-out.
- [ ] Handle path collisions with explicit policy (`error`, `overwrite`, `suffix`).

### User Experience

- [ ] Keep one main config file (`hclforge.yaml`) for defaults/aliases/recipes.
- [ ] Add short intent commands (`add`, `remove`, `set`, `run`).
- [ ] Add `plan`/`dry-run` command for preview-only execution.
- [ ] Add optional interactive mode for guided edits.

### Testing + Quality

- [ ] Add golden tests focused on minimal source drift.
- [ ] Add fixtures with comments, heredocs, nested blocks, and objects.
- [ ] Add command tests for `render`, `replace`, and `apply` paths.
- [ ] Add CI checks for test, lint, and formatting.

## Minimal Required Dependencies (v1)

- `github.com/hashicorp/hcl/v2` (parsing + safe HCL editing via subpackages)
- `gopkg.in/yaml.v3` (YAML config/playbook loading)

CLI note: use Go's standard `flag` package initially to keep dependencies minimal.

## Status

This branch is currently focused on setting up architecture and workflow. Use the checklist above as the source of truth for progress.
