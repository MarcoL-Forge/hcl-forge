# hcl-forge Commands, Operations, and Syntax

This guide covers all supported command-line usage, playbook schema, and edit operation syntax.

## Command List

- `hcl-forge plan -config <playbook.yaml>`
- `hcl-forge apply -config <playbook.yaml>`
- `hcl-forge version`
- `hcl-forge help [command]`

## Global Logging Flags

`plan` and `apply` support:

- `-verbose`
- `-log-level` (`debug|info|warn|error`)
- `-log-format` (`text|json`)
- `-log-output` (`stderr|stdout|<file path>`)
- `-log-artifact <path>`
- `-log-redact <comma-separated-keys>`
- `-quiet`

## Playbook Schema

```yaml
version: 1

input:
  root_dir: .
  files:
    - main.tf

output:
  mode: overwrite # overwrite | target_dir
  target_dir: ./out
  file_map:
    main.tf: renamed-main.tf

options:
  workers: 4
  fail_on_no_change: false

edits:
  - type: search_replace
    old: old-value
    new: new-value
```

## Output Routing

- `output.mode: overwrite` writes changes back to input file paths.
- `output.mode: target_dir` writes under `output.target_dir`.
- `output.file_map` optionally remaps specific `input.files` entries to custom output file names/paths.

Example:

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

- `file_map` keys must exactly match entries in `input.files`.
- `file_map` is only valid when `mode: target_dir`.

## Selector Syntax

Selectors are used by `insert_hcl`, `delete_hcl`, `set_attribute`, and scoped `search_replace`.

### Explicit selector

```yaml
block:
  block_type: resource
  labels: [google_service_account, nodes]
  parents:
    - block_type: module
      labels: [gke_cluster]
```

### Path selector

```yaml
block:
  path: module.gke_cluster.resource.google_service_account.nodes
```

Path grammar supports:

- `resource.<type>.<name>`
- `data.<type>.<name>`
- `module.<name>`
- `module.<type>.<name>` (two-label module pattern)
- `variable.<name>`
- `output.<name>`
- `provider.<name>`
- `locals`
- `terraform`

## Operations

### 1) search_replace

Global text replacement:

```yaml
edits:
  - type: search_replace
    old: rtl-int-
    new: prod-
```

Scoped replacement inside one attribute for a selected block:

```yaml
edits:
  - type: search_replace
    old: rtl-int-
    new: prod-
    block:
      path: module.tfe_workspace.example2
    attribute: name
```

Scoped replacement across all matching blocks:

```yaml
edits:
  - type: search_replace
    old: rtl-int-
    new: prod-
    block:
      block_type: module
      labels: [tfe_workspace, example*]
    attribute: name
```

`match_mode` options for search pattern:

- `glob` (default): literal replacement, plus wildcard pattern replacement when `old` uses glob tokens (`*`, `?`, `[]`).
- `regex`: regular expression replacement.

Regex example:

```yaml
edits:
  - type: search_replace
    old: rtl-int-|01
    new: ''
    match_mode: regex
    block:
      block_type: module
      labels: [tfe_workspace, example.*]
    attribute: name
```

For multiple different substitutions, chain multiple `search_replace` edits in order.

### 2) insert_hcl

Insert at root:

```yaml
edits:
  - type: insert_hcl
    hcl: |
      terraform {
        required_version = ">= 1.5.0"
      }
```

Insert into target block:

```yaml
edits:
  - type: insert_hcl
    block:
      block_type: resource
      labels: [google_storage_bucket, bucket]
    hcl: |
      force_destroy = true
```

Ensure and guard behavior:

```yaml
edits:
  - type: insert_hcl
    ensure_target_block: true
    guard:
      if_target_missing: true
    block:
      path: resource.google_container_node_pool.pool.node_config.shielded_instance_config
    hcl: |
      enable_secure_boot = true
```

### 3) delete_hcl

Delete an attribute:

```yaml
edits:
  - type: delete_hcl
    block:
      block_type: resource
      labels: [google_storage_bucket, bucket]
    attribute: location
```

Delete a whole block:

```yaml
edits:
  - type: delete_hcl
    block:
      path: module.tfe_workspace.example2
```

Delete all matches:

```yaml
edits:
  - type: delete_hcl
    attribute: enable_*
    delete_all: true
```

Keep-only inverse selection:

```yaml
edits:
  - type: delete_hcl
    keep_only: true
    match_mode: regex
    block:
      block_type: module
      labels: [tfe_workspace, example(1|3)]
```

`match_mode` supports `glob` (default) and `regex`.

### 4) set_attribute

Set inside a selected block:

```yaml
edits:
  - type: set_attribute
    block:
      path: resource.google_storage_bucket.bucket
    attribute: location
    value_hcl: '"us-central1"'
```

Create if missing:

```yaml
edits:
  - type: set_attribute
    block:
      block_type: resource
      labels: [google_storage_bucket, bucket]
    attribute: force_destroy
    value_hcl: true
    create_if_missing: true
```

## Environment Variable Interpolation

Supported in playbooks:

- `${VAR}` required variable
- `${VAR:-default}` optional with default

Example:

```yaml
input:
  root_dir: ${HCLFORGE_INPUT_ROOT}
  files:
    - ${HCLFORGE_TARGET_FILE:-storage_bucket.tf}

output:
  mode: target_dir
  target_dir: ${HCLFORGE_OUTPUT_DIR:-./out/pipeline}
```

## Practical Run Examples

```bash
go run ./cmd/hcl-forge plan -config examples/easy/playbook.yaml
go run ./cmd/hcl-forge apply -config examples/easy/playbook.yaml

go run ./cmd/hcl-forge plan -config examples/medium/playbook.yaml
go run ./cmd/hcl-forge apply -config examples/medium/playbook.yaml

go run ./cmd/hcl-forge plan -config examples/hard/playbook.yaml
go run ./cmd/hcl-forge apply -config examples/hard/playbook.yaml

go run ./cmd/hcl-forge apply -config examples/keep-only-demo/playbook.yaml
```

## Troubleshooting

Check active binary and embedded module metadata:

```bash
which -a hcl-forge
go version -m "$(which hcl-forge)"
```

Install a pinned version if `@latest` appears stale:

```bash
go install github.com/MarcoL-Forge/hcl-forge@v0.9.2
```
