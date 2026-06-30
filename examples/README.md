# Examples

This folder contains standalone Terraform inputs and playbooks for three complexity levels:

- `easy`: single-file edits using all operations.
- `medium`: nested block targeting with path selectors and guard/ensure logic.
- `hard`: multi-file edits with target-dir output and mixed selector styles.
- `all-features`: focused playbooks that showcase every supported feature individually.

Run from repository root:

```bash
go run ./cmd/hcl-forge plan -config examples/easy/playbook.yaml
go run ./cmd/hcl-forge apply -config examples/easy/playbook.yaml

go run ./cmd/hcl-forge plan -config examples/medium/playbook.yaml
go run ./cmd/hcl-forge apply -config examples/medium/playbook.yaml

go run ./cmd/hcl-forge plan -config examples/hard/playbook.yaml
go run ./cmd/hcl-forge apply -config examples/hard/playbook.yaml

go run ./cmd/hcl-forge plan -config examples/all-features/playbook-search-replace.yaml
go run ./cmd/hcl-forge plan -config examples/all-features/playbook-insert.yaml
go run ./cmd/hcl-forge plan -config examples/all-features/playbook-insert-guard-ensure.yaml
go run ./cmd/hcl-forge plan -config examples/all-features/playbook-delete.yaml
go run ./cmd/hcl-forge plan -config examples/all-features/playbook-delete-all.yaml
go run ./cmd/hcl-forge plan -config examples/all-features/playbook-delete-wildcard.yaml
go run ./cmd/hcl-forge plan -config examples/all-features/playbook-set-attribute.yaml
go run ./cmd/hcl-forge plan -config examples/all-features/playbook-selector-compat-and-output-overwrite.yaml
```

`all-features` playbook coverage:

- `playbook-search-replace.yaml`: `search_replace` across multiple values.
- `playbook-insert.yaml`: root-level and block-targeted `insert_hcl`.
- `playbook-insert-guard-ensure.yaml`: `ensure_target_block` and `guard` options.
- `playbook-delete.yaml`: attribute and block deletion.
- `playbook-delete-all.yaml`: `delete_all` for attributes and blocks.
- `playbook-delete-wildcard.yaml`: wildcard selectors (`*`) for `delete_hcl` blocks and attribute names.
- `playbook-set-attribute.yaml`: `set_attribute` update and `create_if_missing`.
- `playbook-selector-compat-and-output-overwrite.yaml`: backward-compatible selector keys (`type`) and `output.mode=overwrite`.
