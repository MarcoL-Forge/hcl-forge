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