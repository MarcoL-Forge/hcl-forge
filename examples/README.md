# Examples

This folder contains standalone Terraform inputs and playbooks for three complexity levels:

- `easy`: single-file edits using all operations.
- `medium`: nested block targeting with path selectors and guard/ensure logic.
- `hard`: multi-file edits with target-dir output and mixed selector styles.

Run from repository root:

```bash
go run ./cmd/hcl-forge plan -config examples/easy/playbook.yaml
go run ./cmd/hcl-forge apply -config examples/easy/playbook.yaml

go run ./cmd/hcl-forge plan -config examples/medium/playbook.yaml
go run ./cmd/hcl-forge apply -config examples/medium/playbook.yaml

go run ./cmd/hcl-forge plan -config examples/hard/playbook.yaml
go run ./cmd/hcl-forge apply -config examples/hard/playbook.yaml
```
