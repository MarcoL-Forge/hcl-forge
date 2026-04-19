# hcl-forge Packages

This file is the implementation-oriented companion to [README.md](README.md).
It should be updated whenever package-level functionality is added, changed, or removed.

## Current Status

The project currently has working package-level functionality in these areas:

- `cmd/hcl-forge`: executable entrypoint
- `internal/cli`: command parsing and user-facing command workflows
- `internal/document`: document loading, rendering, and writing
- `internal/parser`: HCL2-backed targeted attribute replacement
- `internal/playbook`: YAML playbook loading and path resolution

The following packages exist but are still placeholders:

- `internal/index`

## Package Map

### `cmd/hcl-forge`

Purpose:
- build the `hcl-forge` executable
- delegate command handling to the internal CLI package

Current functionality:
- calls `cli.Run(os.Args[1:])`
- exits with a non-zero status on command errors

### `internal/cli`

Purpose:
- expose user-facing commands and flags
- translate CLI input into document, parser, and playbook operations

Current functionality:
- `render`
  - loads a Terraform or HCL file
  - renders the current document bytes
  - writes to `--out` and/or prints to `--stdout`
- `replace`
  - performs one targeted attribute replacement using either a generic `--target` selector or the legacy `--block-type` / `--labels` / `--attr` flags
  - supports scalar replacements and raw HCL expression replacements through `--value-type`
  - writes to `--out` and/or prints to `--stdout`
- `apply`
  - loads a YAML playbook
  - applies the listed operations in order
  - writes to playbook `output`, CLI `--out`, and/or prints to `--stdout`
- `help`
  - prints usage text for the current commands

Current limitations:
- `replace` handles top-level attributes, top-level block attributes, and nested unlabeled block attributes
- selector resolution can fail when a selector is ambiguous
- `apply` currently supports only `set_attribute` operations
- nested object and full block manipulation are not implemented yet

### `internal/document`

Purpose:
- represent a source file in memory
- own exact source bytes for round-trip workflows
- handle basic file I/O

Current functionality:
- `Document`
  - stores file `Path`
  - stores raw file bytes in `Raw`
- `LoadDocument`
  - loads a file from disk into a `Document`
- `RenderDocument`
  - returns the current document bytes unchanged
- `WriteDocument`
  - writes a document to any requested output path
  - creates missing parent directories automatically

Design note:
- this package should remain byte-oriented
- it should not own Terraform semantics or HCL parsing logic

### `internal/parser`

Purpose:
- use HCL2 to understand and modify Terraform/HCL structure

Current functionality:
- `ReplaceAttributeInput`
  - describes a targeted attribute replacement using either a selector or the legacy block fields
- `ReplaceAttributeValue`
  - parses the document with `hclwrite.ParseConfig`
  - resolves a target body from a top-level attribute, a top-level block, or a nested block selector
  - supports selector-based targeting and legacy block-type targeting in the same API
  - replaces or sets an attribute value with either a cty scalar or raw HCL tokens
  - returns a new `Document` with the updated bytes

Supported value types:
- `string`
- `bool`
- `number`
- `hcl`

Current limitations:
- nested labeled blocks are not supported in selectors yet
- selector parsing is dot-delimited, so it is intentionally simple and does not yet model full Terraform addresses
- no block addition or block removal yet

### `internal/playbook`

Purpose:
- load declarative transformation instructions from YAML

Current functionality:
- `Playbook`
  - `version`
  - `input`
  - `output`
  - `operations`
- `Operation`
  - `op`
  - `target`
  - `block_type`
  - `labels`
  - `attribute`
  - `value`
  - `value_type`
- `Load`
  - reads YAML playbooks
  - defaults `version` to `1` when omitted
  - validates required fields such as `input` and `operations`
  - resolves relative `input` and `output` paths from the playbook directory

Current limitations:
- one input file per playbook
- only `set_attribute` operations are usable through the current CLI
- legacy `block_type` / `labels` / `attribute` targeting remains supported for compatibility, but `target` is the preferred shape for new playbooks

### `internal/index`

Purpose:
- eventually provide block and attribute lookup helpers on top of parsed HCL

Current functionality:
- package placeholders only

Planned functionality:
- block indexing
- attribute indexing
- address-based lookups
- source range lookup support

## CLI Commands

Current commands:

- `hcl-forge render --in <file> [--out <file>] [--stdout]`
- `hcl-forge replace --in <file> [--target <selector> | --block-type <type> [--labels <a,b>] --attr <name>] --value <value> [--value-type <type>] [--out <file>] [--stdout]`
- `hcl-forge apply --playbook <file> [--out <file>] [--stdout]`

Selector examples:

- top-level tfvars attribute: `gcp_region`
- top-level block attribute: `module.network.source`
- Terraform resource attribute: `resource.google_container_cluster.this.name`
- nested block attribute: `resource.google_container_cluster.this.node_config.service_account`

## Playbook Schema

Current example:

```yaml
version: 1
input: testing/gke/main.tf
output: output.tf

operations:
  - op: set_attribute
    target: resource.google_container_cluster.this.name
    value: example-cluster
    value_type: string
```

For raw lists, maps, or objects, use `value_type: hcl`.

## Update Rule

When functionality changes, update this file with:

- new exported package functions
- new commands or flags
- new playbook operations
- new limitations removed or introduced
- package responsibilities if they shift