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