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

## Examples

Sample Terraform inputs and playbooks are in `examples/` with three complexity levels:

- `easy`: single-file edits using all operations
- `medium`: nested selectors with guard/ensure behavior
- `hard`: multi-file edits with target-dir output and mixed selector styles

Run from repo root:

```bash
go run ./cmd/hcl-forge plan -config examples/easy/playbook.yaml
go run ./cmd/hcl-forge apply -config examples/easy/playbook.yaml

go run ./cmd/hcl-forge plan -config examples/medium/playbook.yaml
go run ./cmd/hcl-forge apply -config examples/medium/playbook.yaml

go run ./cmd/hcl-forge plan -config examples/hard/playbook.yaml
go run ./cmd/hcl-forge apply -config examples/hard/playbook.yaml
```

## Pre-commit Quality Checks

The repository uses native `pre-commit` hooks for formatting, linting, and quality checks.

Install once per clone:

```bash
git config --unset-all core.hooksPath || true
pre-commit install
```

Run all hooks manually:

```bash
pre-commit run --all-files
```

What runs before every commit:

- trailing-whitespace check for Go/YAML/Markdown files
- `gofmt -w` for Go files
- `go vet ./...`
- `golangci-lint run ./...` (pinned to `v1.64.8`)

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
go run ./cmd/hcl-forge plan -config examples/medium/playbook.yaml
go run ./cmd/hcl-forge apply -config examples/medium/playbook.yaml
```

Notes:

- Matching is exact on `block_type` and label order.
- If multiple blocks match, the first match found is used.
- Block snippets are convergent: rerunning the same `insert_hcl` merges into matching child blocks instead of appending duplicates.

### Rigorous Selector Model (Recommended Reading)

`hcl-forge` supports two selector styles for `insert_hcl`, `delete_hcl`, and `set_attribute`.

1. Explicit selector style:

```yaml
block:
	block_type: resource
	labels: [google_service_account, nodes]
	parents:
		- block_type: module
			labels: [gke_cluster]
```

2. Dot path selector style:

```yaml
block:
	path: module.gke_cluster.resource.google_service_account.nodes
```

Both styles work. Dot path is compiled into the explicit selector model internally.

You can mix styles across edits in the same playbook (for example, `insert_hcl` using explicit selectors and `delete_hcl` using `block.path`).

`insert_hcl` also supports guarded and ensure semantics:

- `ensure_target_block: true`: create missing target block path before inserting snippet entries.
- `guard.if_target_exists: true`: apply only when target block already exists.
- `guard.if_target_missing: true`: apply only when target block is missing.

Guarded + ensure example:

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

Strict rules to avoid ambiguity:

- If `block.path` is set, do not set `block_type`/`type`, `labels`, or `parents` in the same selector.
- If `block.path` is not set, use explicit fields (`block_type` + `labels`, optional `parents`).
- Matching is deterministic and exact (type + label order + parent chain).
- `guard.if_target_exists` and `guard.if_target_missing` cannot both be true.
- `ensure_target_block` and `guard` require a `block` selector.

Mixed-style playbook example:

```yaml
edits:
	- type: insert_hcl
		block:
			block_type: resource
			labels: [google_service_account, nodes]
		hcl: |
			description = "managed"

	- type: delete_hcl
		block:
			path: resource.google_service_account.nodes
		attribute: description
```

#### Terraform Alignment

The dot syntax is Terraform-like for block addressing and AST traversal.

- Top-level block paths mirror Terraform concepts:
	- `resource.google_service_account.nodes`
	- `data.google_client_config.default`
	- `module.gke_cluster`
	- `provider.google`
- Nested segments (for example `.node_config.shielded_instance_config`) represent AST block traversal.

It does **not** currently represent arbitrary expression/object traversal such as `locals.foo.bar[0]` inside values.

#### Path Grammar

Supported path segments:

- `resource.<type>.<name>`
- `data.<type>.<name>`
- `module.<name>`
- `variable.<name>`
- `output.<name>`
- `provider.<name>`
- `locals`
- `terraform`
- Any additional segment after those is treated as nested block traversal.

Examples:

```yaml
# Exact equivalent selectors
block:
	path: resource.google_service_account.nodes

# equals
block:
	block_type: resource
	labels: [google_service_account, nodes]

# Nested path
block:
	path: resource.google_container_node_pool.pool.node_config.shielded_instance_config
```

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
- when a targeted `block` is missing for attribute deletion, the edit is a no-op (idempotent) instead of a hard failure

## Set Attribute Edits

Use `set_attribute` for selector-scoped AST updates without text search/replace.

```yaml
edits:
	- type: set_attribute
		block:
			path: resource.google_storage_bucket.bucket
		attribute: location
		value_hcl: '"us-central1"'

	- type: set_attribute
		block:
			block_type: resource
			labels: [google_storage_bucket, bucket]
		attribute: force_destroy
		value_hcl: true
		create_if_missing: true
```

Behavior:

- If the target block is missing, the edit is a no-op.
- If the attribute already has the same expression, the edit is a no-op.
- If `create_if_missing: true`, missing attributes are created; otherwise the edit is a no-op.

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

go run ./cmd/hcl-forge plan -config examples/hard/playbook.yaml
go run ./cmd/hcl-forge apply -config examples/hard/playbook.yaml
```

Harness mapping example (set generic vars from Harness expressions):

```bash
export HCLFORGE_INPUT_ROOT="<+pipeline.variables.inputRoot>"
export HCLFORGE_TARGET_FILE="<+pipeline.variables.targetFile>"
export HCLFORGE_OUTPUT_DIR="./out/<+pipeline.sequenceId>"
export HCLFORGE_PIPELINE_NAME="<+pipeline.name>"
export HCLFORGE_ENV="<+pipeline.variables.environment>"

go run ./cmd/hcl-forge plan -config examples/hard/playbook.yaml
go run ./cmd/hcl-forge apply -config examples/hard/playbook.yaml
```

This keeps playbooks platform-neutral while still allowing Harness to populate values.

See `examples/hard/playbook.yaml` for a complete template-ready playbook.

## Logging and Debug Artifacts

`hcl-forge` provides structured logging for both local runs and CI pipelines.

Supported flags on both `plan` and `apply`:

- `--verbose`: shortcut for debug-level logs
- `--log-level`: `debug|info|warn|error`
- `--log-format`: `text|json`
- `--log-output`: `stderr|stdout|<file-path>`
- `--log-artifact`: optional NDJSON artifact file for pipeline ingestion
- `--log-redact`: comma-separated additional keys to redact (defaults already include common secret keys)
- `--quiet`: suppress human-readable summary output and emit structured logs only

Examples:

```bash
# Human-readable local logs
go run ./cmd/hcl-forge plan \
	-config examples/medium/playbook.yaml \
	--verbose \
	--log-format text \
	--log-output stderr

# Pipeline-friendly JSON logs + NDJSON artifact
go run ./cmd/hcl-forge apply \
	-config examples/hard/playbook.yaml \
	--log-level info \
	--log-format json \
	--log-output stdout \
	--log-artifact ./out/hclforge-events.ndjson \
	--log-redact session_token,db_password \
	--quiet
```

Debug coverage includes:

- command lifecycle (`plan_start`, `apply_start`, completion/failure)
- per-file worker lifecycle (`file_job_start`, `file_job_completed`, `file_job_failed`)
- per-edit timing (`edit_start`, `edit_completed`, `edit_failed`)
- selector resolution traces for `insert_hcl`

Every structured log event includes:

- `schema_version` (currently `hclforge.log.v1`)
- monotonic `event_id` per process run

## Publish and Install

This project can be published as downloadable binaries using GitHub Releases.

### User installs

Install from source (requires Go):

```bash
go install github.com/Marc0l95/hclforge/cmd/hcl-forge@latest
```

Install prebuilt binaries:

1. Go to GitHub Releases for this repository.
2. Download the archive matching your OS/CPU (for example `hcl-forge_v1.2.3_darwin_arm64.tar.gz`).
3. Extract and place `hcl-forge` in your `PATH`.

### Maintainer release flow

Release workflows are currently **manual-only** (`workflow_dispatch`) for controlled rollout.

Production release files:

- `.github/workflows/semantic-release.yml`: computes next semantic version from Conventional Commits and creates/pushes a `vX.Y.Z` tag.
- `.github/workflows/release.yml`: runs GoReleaser to publish release artifacts.
- `.goreleaser.yaml`: build/archive/changelog configuration.

Recommended release runbook:

1. Ensure `main` is green (`tests` and `Security Checks` workflows passing).
2. Run `Semantic Release (Conventional Commits)` manually from GitHub Actions.
3. Verify the new `vX.Y.Z` tag exists.
4. Run `release` workflow manually, selecting the new tag as the ref.

Conventional Commit bump rules used by semantic release:

- major: `BREAKING CHANGE` footer or `!` in commit type
- minor: `feat:`
- patch: `fix:` or `perf:`
- no release: commits that do not match release rules

GoReleaser publishes:

- Linux/macOS/Windows binaries (`amd64`, `arm64`)
- release archives and checksums
- GitHub Release assets

### Security and Supply Chain Posture

Security controls currently in place:

- `.github/workflows/security-checks.yml` runs on pull requests and pushes to `main`.
- Blocking scans:
	- `govulncheck` for Go dependency/runtime vulnerabilities
	- `gitleaks` for secret detection
	- Trivy container scan (`HIGH`, `CRITICAL`) against built image
- Workflow actions are pinned to immutable commit SHAs.
- `.github/dependabot.yml` enables weekly updates for:
	- Go modules (`gomod`)
	- GitHub Actions

Operational recommendation for production:

- Require `tests` and `Security Checks` as required status checks in branch protection before merging to `main`.

### Install in Go pipelines

Use `go install` (preferred over `go get` for binaries):

```bash
go install github.com/Marc0l95/hclforge/cmd/hcl-forge@latest
```

Pinned version:

```bash
go install github.com/Marc0l95/hclforge/cmd/hcl-forge@v0.1.0
```

### Docker Image for Pipeline Ingestion

Release publishing is currently focused on GoReleaser binaries. Docker image distribution is a separate, optional path you can run from your own CI.

Build an image from this repository:

```bash
docker build -t hcl-forge:local .
```

Run `hcl-forge` in a container against your checked out repository:

```bash
docker run --rm \
	-v "$PWD:/work" \
	-w /work \
	-e HCLFORGE_INPUT_ROOT=./testing/gke \
	-e HCLFORGE_TARGET_FILE=storage_bucket.tf \
	-e HCLFORGE_OUTPUT_DIR=./out/pipeline \
	hcl-forge:local plan -config examples/hard/playbook.yaml

docker run --rm \
	-v "$PWD:/work" \
	-w /work \
	-e HCLFORGE_INPUT_ROOT=./testing/gke \
	-e HCLFORGE_TARGET_FILE=storage_bucket.tf \
	-e HCLFORGE_OUTPUT_DIR=./out/pipeline \
	hcl-forge:local apply -config examples/hard/playbook.yaml
```

If you want to use GAR for Harness + Aqua scanning, push this image from your CI:

```bash
docker tag hcl-forge:local ${GAR_LOCATION}-docker.pkg.dev/${GCP_PROJECT_ID}/${GAR_REPOSITORY}/hcl-forge:latest
docker push ${GAR_LOCATION}-docker.pkg.dev/${GCP_PROJECT_ID}/${GAR_REPOSITORY}/hcl-forge:latest
```

Then configure Harness to pull that image, run Aqua scan, and execute `hcl-forge` commands in the step.