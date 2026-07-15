# hcl-forge Commands and Examples

This guide contains the operational command list and practical examples for day-to-day usage.

## Command List

- `hcl-forge plan -config <playbook.yaml>`
  - Dry run. Computes and prints edit results without writing files.
- `hcl-forge apply -config <playbook.yaml>`
  - Applies edits and writes output files.
- `hcl-forge version`
  - Prints the running CLI version.
- `hcl-forge help [command]`
  - Shows command help.

## Global Logging Flags

`plan` and `apply` support:

- `-verbose`
- `-log-level` (`debug|info|warn|error`)
- `-log-format` (`text|json`)
- `-log-output` (`stderr|stdout|<file path>`)
- `-log-artifact <path>`
- `-log-redact <comma-separated-keys>`
- `-quiet`

## Common Usage Patterns

### Plan then apply

```bash
hcl-forge plan -config examples/easy/playbook.yaml
hcl-forge apply -config examples/easy/playbook.yaml
```

### JSON logs for CI

```bash
hcl-forge apply \
  -config examples/hard/playbook.yaml \
  -log-format json \
  -log-output stdout \
  -quiet
```

### Write NDJSON artifact

```bash
hcl-forge plan \
  -config examples/hard/playbook.yaml \
  -log-artifact ./out/hclforge-events.ndjson
```

## Example Playbooks

### Easy

```bash
go run ./cmd/hcl-forge plan -config examples/easy/playbook.yaml
go run ./cmd/hcl-forge apply -config examples/easy/playbook.yaml
```

### Medium

```bash
go run ./cmd/hcl-forge plan -config examples/medium/playbook.yaml
go run ./cmd/hcl-forge apply -config examples/medium/playbook.yaml
```

### Hard

```bash
go run ./cmd/hcl-forge plan -config examples/hard/playbook.yaml
go run ./cmd/hcl-forge apply -config examples/hard/playbook.yaml
```

### Keep-only demo

```bash
go run ./cmd/hcl-forge apply -config examples/keep-only-demo/playbook.yaml
```

## Output Routing Example

Use `output.mode: target_dir` for directory routing and `output.file_map` for file renaming.

```yaml
input:
  root_dir: ./terraform
  files:
    - main.tf
    - modules/gke/cluster.tf

output:
  mode: target_dir
  target_dir: ./out
  file_map:
    main.tf: generated/root.tf
    modules/gke/cluster.tf: generated/platform/gke-cluster-prod.tf
```

Rules:

- `output.file_map` keys must match `input.files` entries exactly.
- `output.file_map` is valid only with `output.mode: target_dir`.
- Files not mapped keep default relative output paths.

## Selector Styles

Both are supported for `insert_hcl`, `delete_hcl`, and `set_attribute`.

### Explicit selector

```yaml
block:
  block_type: resource
  labels: [google_service_account, nodes]
```

### Path selector

```yaml
block:
  path: resource.google_service_account.nodes
```

## Troubleshooting

- If `hcl-forge version` shows an unexpected value, check active binary path:

```bash
which -a hcl-forge
go version -m "$(which hcl-forge)"
```

- If install via `@latest` appears stale, install a pinned tag:

```bash
go install github.com/MarcoL-Forge/hcl-forge@v0.9.2
```
