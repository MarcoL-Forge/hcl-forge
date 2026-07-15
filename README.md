# hcl-forge

hcl-forge is a Terraform/HCL bulk editor for local workflows and CI pipelines.
It applies declarative YAML playbooks to one or many files with deterministic, reviewable output.

## Why hcl-forge

- Safe, repeatable HCL transformations at scale.
- Plan-before-apply workflow for confidence.
- Works the same way locally and in CI.
- Supports selector-driven edits and output routing.

## Install

Use these install commands:

- Local development install (from this repo):

```bash
go install .
```

- Latest release:

```bash
go install github.com/MarcoL-Forge/hcl-forge@latest
```

- Version-pinned release:

```bash
go install github.com/MarcoL-Forge/hcl-forge@v0.9.2
```

Verify:

```bash
hcl-forge version
```

## Commands

Available commands:

- `hcl-forge plan`
- `hcl-forge apply`
- `hcl-forge version`
- `hcl-forge help`

For full command list, flags, and usage examples, see:

- `docs/commands-and-examples/README.md`

## Quick Start

```bash
hcl-forge plan -config examples/easy/playbook.yaml
hcl-forge apply -config examples/easy/playbook.yaml
```

## Project Layout

- `cmd/hcl-forge`: CLI entrypoint
- `internal/`: core implementation
- `examples/`: sample playbooks and fixtures
- `testing/`: larger test fixtures

## Testing

```bash
make test-unit
make test-integration
make test-e2e
```

## Release and CI

Release automation and security workflows are configured in `.github/workflows/`.
GoReleaser configuration is in `.goreleaser.yaml`.

## License

See `LICENSE`.