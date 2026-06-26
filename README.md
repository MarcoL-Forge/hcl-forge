# hcl-forge

`hcl-forge` is a Go-based CLI and library for safely modifying Terraform and HCL files at scale.

It is designed for local developer workflows and CI/CD pipelines where teams need deterministic, reviewable, and repeatable transformations across one or many `.tf`, `.tfvars`, or `.hcl` files while preserving existing formatting and comments as much as possible.

## Goals

`hcl-forge` aims to provide a safe editing layer for Terraform/HCL source files.

Core goals:

- Read one or many Terraform/HCL files.
- Apply declarative edit operations from a YAML playbook.
- Support local and pipeline execution using the same workflow.
- Preserve existing formatting, comments, and file structure where possible.
- Provide a `plan` mode to preview changes before writing.
- Support deterministic input/output routing for generated or patched files.
- Keep the CLI dependency-light and automation-friendly.

## Non-goals

`hcl-forge` is not intended to replace Terraform.

It does not:

- Execute Terraform plans or applies.
- Validate provider-specific resource schemas.
- Guarantee semantic Terraform correctness.
- Replace `terraform fmt` or `terraform validate`.

Recommended pipeline flow:

```bash
hcl-forge plan -config hclforge.yaml
hcl-forge apply -config hclforge.yaml
terraform fmt
terraform validate
```

## Test Organization

Use package-local tests by default, and separate layers using build tags rather than top-level test folders.

- Unit tests: keep `*_test.go` next to the code they test (fast `go test ./...`).
- Integration tests: keep near the package under test and gate with `//go:build integration`.
- E2E tests: place near the CLI entrypoint and gate with `//go:build e2e`.
- Fixtures: keep reusable sample files in `testing/` or package-scoped `testdata/` directories.

This repo follows that model, which is idiomatic in Go and works well in CI.

Run all layers explicitly:

```bash
make test-unit
make test-integration
make test-e2e
make test-coverage
```

`make test-coverage` writes a coverage profile to `coverage.out` and prints the overall statement coverage percentage.

## Insert HCL Edits

Use `insert_hcl` to insert Terraform attributes or blocks at the file root or inside a specific block.

```yaml
edits:
	- type: insert_hcl
		block:
			block_type: resource
			labels:
				- google_storage_bucket
				- bucket
		hcl: |
			force_destroy = true

	- type: insert_hcl
		hcl: |
			terraform {
				required_version = ">= 1.5.0"
			}
```

- When `block` is provided, insertion happens inside the first matching block (`block_type` + exact `labels`).
- `block.type` is still accepted for backward compatibility.
- When `block` is omitted, insertion happens at the root body of the file.

### User Guide: Targeting Blocks

Use `block.block_type` to choose the Terraform block kind, and `block.labels` to choose the exact block instance.

For a Terraform resource:

```hcl
resource "google_container_node_pool" "this" {
	# ...
}
```

Target selector:

```yaml
block:
	block_type: resource
	labels: [google_container_node_pool, this]
```

For nested unlabeled blocks (for example `node_config {}`), use empty labels:

```yaml
block:
	block_type: node_config
	labels: []
```

Complete example (insert inside `node_config`):

```yaml
edits:
	- type: insert_hcl
		block:
			block_type: node_config
			labels: []
		hcl: |
			disk_size_gb = 100
			tags = ["gke-node", "secure"]

			shielded_instance_config {
				enable_secure_boot          = true
				enable_integrity_monitoring = true
			}
```

Run it:

```bash
go run ./cmd/hcl-forge plan -config example_playbook/tf_insert_node_config.yaml
go run ./cmd/hcl-forge apply -config example_playbook/tf_insert_node_config.yaml
```

Notes:

- Matching is exact on `block_type` and label order.
- If multiple blocks match, the first match found is used.

## Delete HCL Edits

Use `delete_hcl` to remove either:

- an attribute (key/value), optionally inside a selected block
- a whole block selected by `block_type` + `labels`

Delete an attribute inside a resource:

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
			block_type: variable
			labels: [project_id]
```

Delete a root-level attribute (no `block` selector):

```yaml
edits:
	- type: delete_hcl
		attribute: required_version
```

Delete all matching attributes or blocks across the file:

```yaml
edits:
	- type: delete_hcl
		attribute: oauth_scopes
		delete_all: true

	- type: delete_hcl
		block:
			block_type: management
			labels: []
		delete_all: true
```

`delete_all: true` behavior:

- with `attribute`: removes every matching attribute (scope is root + nested blocks, or only matched blocks when `block` is provided)
- with `block`: removes every matching block
- without `delete_all` (default): removes only the first match

## CI Pipeline Inputs

Playbooks support environment variable interpolation during config load.

In any CI run step (including Harness), `hcl-forge` reads environment variables from the process and automatically substitutes them into playbooks when loading the config. No separate templating step is required.

Supported syntax:

- `${VAR}`: required environment variable (load fails if missing)
- `${VAR:-default}`: optional variable with fallback default

Example playbook pattern:

```yaml
input:
	root_dir: ${HCLFORGE_INPUT_ROOT}
	files:
		- ${HCLFORGE_TARGET_FILE:-storage_bucket.tf}

output:
	mode: target_dir
	target_dir: ${HCLFORGE_OUTPUT_DIR:-./out/pipeline}
```

Generic CI step example:

```bash
export HCLFORGE_INPUT_ROOT="./testing/gke"
export HCLFORGE_TARGET_FILE="storage_bucket.tf"
export HCLFORGE_OUTPUT_DIR="./out/pipeline"
export HCLFORGE_PIPELINE_NAME="${HCLFORGE_PIPELINE_NAME:-my-pipeline}"
export HCLFORGE_ENV="${HCLFORGE_ENV:-dev}"

go run ./cmd/hcl-forge plan -config example_playbook/tf_harness_pipeline.yaml
go run ./cmd/hcl-forge apply -config example_playbook/tf_harness_pipeline.yaml
```

Harness mapping example (set generic vars from Harness expressions):

```bash
export HCLFORGE_INPUT_ROOT="<+pipeline.variables.inputRoot>"
export HCLFORGE_TARGET_FILE="<+pipeline.variables.targetFile>"
export HCLFORGE_OUTPUT_DIR="./out/<+pipeline.sequenceId>"
export HCLFORGE_PIPELINE_NAME="<+pipeline.name>"
export HCLFORGE_ENV="<+pipeline.variables.environment>"

go run ./cmd/hcl-forge plan -config example_playbook/tf_harness_pipeline.yaml
go run ./cmd/hcl-forge apply -config example_playbook/tf_harness_pipeline.yaml
```

This keeps playbooks platform-neutral while still allowing Harness to populate values.

See `example_playbook/tf_harness_pipeline.yaml` for a complete template-ready playbook.