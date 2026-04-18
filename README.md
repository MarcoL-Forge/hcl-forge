# hcl-forge
hclforge — A CLI tool for transforming HCL/Terraform files from templates. Define value replacements, block additions, removals, and conditional logic in a YAML spec — output is plain, ready-to-use .tf files.

## Goal

The long-term goal of `hcl-forge` is to ingest Terraform and HCL files, understand their structure, manipulate Terraform objects safely, and write updated files back out while preserving untouched formatting and comments.

That means the tool needs to support operations such as:

- changing attribute values
- removing attributes or blocks
- adding new blocks or attributes
- targeting Terraform objects by logical address instead of raw line numbers
- writing output files with minimal source drift

## Design Principles

The design should optimize for round-trip editing rather than full file regeneration.

Key principles:

- Preserve untouched source bytes exactly.
- Parse once and build indexes for reuse.
- Analyze with `hclsyntax`, but do not use the parse tree as the write format.
- Generate new HCL snippets with `hclwrite` only when needed.
- Apply narrow source edits against byte ranges instead of re-rendering entire files.
- Treat Terraform objects as addressable entities such as `resource.aws_instance.web` or `module.network`.

## Why Not Just Rewrite The AST?

Rebuilding a whole file from an AST is easy to make valid, but it is not good at preserving original formatting, comments, or the user's preferred layout.

For this project, the correct mental model is:

- `hclsyntax` is for understanding structure
- `hclwrite` is for rendering valid new snippets
- the source file remains the canonical representation

The engine should therefore operate as a round-trip patching system:

1. load original bytes
2. parse structure and tokens
3. locate the exact byte ranges that correspond to Terraform objects
4. generate minimal edits
5. write a new output file by patching the original source

## Architecture

The recommended implementation uses three representations of the same file:

1. Original source bytes
2. Parsed syntax and token metadata
3. Planned source edits

### 1. Document Layer

Each input file should be modeled as a `Document` containing:

- path
- raw source bytes
- line index
- parse diagnostics
- tokens
- parsed syntax tree
- indexes for blocks and attributes

This layer is responsible for loading `.tf`, `.tfvars`, and eventually generic `.hcl` files.

### 2. Parse Layer

Run two analysis passes over the same source:

- lexical/token pass using `hclsyntax`
- syntax parse pass using `hclsyntax`

Use these passes for:

- block discovery
- attribute discovery
- expression range tracking
- comment and trivia awareness
- exact source byte ranges

### 3. Index Layer

Build indexes that map Terraform objects to source ranges.

Examples:

- block address: `resource.google_container_cluster.main`
- block address: `module.network`
- attribute path: `resource.google_container_cluster.main.location`

Each indexed object should include:

- file path
- start/end byte range
- block type and labels
- attribute name
- parent address

### 4. Edit Layer

All changes should be represented as explicit edits rather than ad hoc string manipulation.

An edit should contain:

- start byte
- end byte
- replacement bytes
- description

Edits should be applied in descending byte order so offsets stay stable.

### 5. Snippet Generation Layer

When a change needs brand new HCL content, generate only that snippet with `hclwrite`.

Examples:

- new attribute assignment
- new nested block
- replacement object value

Then splice the snippet into the original source at the correct insertion point.

### 6. Writer Layer

The writer should:

- apply ordered edits to original bytes
- write the result to an output file
- optionally produce a diff
- optionally run `terraform fmt`

Formatting should be optional because full formatting may change more source than intended.

## Recommended Object Model

The internal API should be centered on Terraform objects rather than files and lines.

Useful core concepts:

- `Document`
- `Module`
- `BlockRef`
- `AttributeRef`
- `Edit`
- `Operation`

Example responsibilities:

- `Document`: source file, tokens, parse result, indexes
- `Module`: a collection of Terraform files analyzed as one logical unit
- `BlockRef`: block metadata and source ranges
- `AttributeRef`: attribute metadata and source ranges
- `Operation`: a planned manipulation such as set attribute or remove block

## Supported Manipulations

The first implementation should focus on narrow, deterministic operations:

- list blocks
- list attributes
- set scalar attribute value
- remove attribute
- remove block
- add top-level block

Later operations can build on the same engine:

- rename block labels
- replace nested objects
- append list members
- rewrite traversals
- module-wide refactors across multiple files

## Package Layout

A clean Go package structure would look like this:

```text
cmd/hcl-forge/
	main.go
internal/document/
	document.go
	lines.go
internal/parser/
	lexer.go
	parse.go
	ranges.go
internal/index/
	blocks.go
	attributes.go
	addresses.go
internal/edit/
	edit.go
	apply.go
	diff.go
internal/ops/
	set_attribute.go
	remove_attribute.go
	add_block.go
	remove_block.go
internal/render/
	snippets.go
	indent.go
internal/terraform/
	model.go
	selectors.go
```

## Recommended Workflow

The core flow should be:

1. Load one file or a full module directory.
2. Parse files into syntax trees and token metadata.
3. Build block and attribute indexes.
4. Resolve Terraform objects by logical address.
5. Plan edits against exact byte ranges.
6. Apply edits to the original bytes.
7. Write the updated source to an output file.
8. Optionally format or diff the result.

## First Milestone

The first useful prototype should do only this:

1. ingest a `.tf` file
2. tokenize it with `hcl/v2`
3. parse blocks and attributes
4. print an inventory of Terraform objects and source ranges
5. perform one safe edit such as setting an attribute value
6. write the result to a new file while preserving untouched comments and formatting

That milestone is enough to validate the core architecture.

## Testing Strategy

This project should use golden tests with real Terraform examples.

Test files should include:

- comments above blocks
- inline comments
- blank lines
- nested blocks
- objects and lists
- heredocs
- mixed indentation

Success should be measured by minimal source drift, not just syntactic validity.

## Implementation Notes

Recommended libraries:

- `github.com/hashicorp/hcl/v2`
- `github.com/hashicorp/hcl/v2/hclsyntax`
- `github.com/hashicorp/hcl/v2/hclwrite`

Guidance:

- use `hclsyntax` for parsing, tokens, and source ranges
- use `hclwrite` for rendering new snippets only
- avoid using Terraform internals until full Terraform semantic analysis is actually needed

## Summary

The intended shape of `hcl-forge` is:

- a Terraform/HCL ingestion engine
- a structural index over blocks and attributes
- a range-based edit engine for round-trip source preservation
- a future YAML-driven transform layer on top

This keeps the engine reusable for CLI commands, automated transforms, and tests while preserving the original authoring style of Terraform files as much as possible.
