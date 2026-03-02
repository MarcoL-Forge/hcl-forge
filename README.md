# hclforge

> Transform HCL/Terraform files from templates using a declarative YAML spec.

hclforge reads existing `.tf` files as templates, applies an ordered set of transformations defined in a `transform.yaml` file, and writes clean, standalone `.tf` files to a target directory. Source files are **never modified**. Output requires no runtime dependency on hclforge.

---

## Features

- **Set, remove, and rename** attributes and blocks in any `.tf` file
- **Copy blocks** from other files, with optional rename and nested changes
- **Add blocks** from a reusable snippet library
- **Conditional logic** via Go `text/template` expressions (`{{ .Flags.x }}`, `{{ not .Flags.x }}`)
- **Variable interpolation** in any value (`{{ .Vars.env }}`)
- **Dry-run mode** — preview changes without writing anything
- **CLI overrides** — override vars and flags at invocation time
- Preserves HCL formatting and comments throughout

---

## Installation

```bash
go install github.com/Marc0l95/hclforge/cmd/hclforge@latest
```

Or build from source:

```bash
git clone https://github.com/Marc0l95/hclforge
cd hclforge
make build
```

---

## Quick start

**1. Organise your project**

```
my-infra/
├── templates/          # source .tf files — never modified
│   ├── main.tf
│   └── variables.tf
├── snippets/           # reusable blocks available to add_block and copy_block
│   └── ebs_volume.tf
└── transform.yaml      # transformation spec
```

**2. Write a spec**

```yaml
source_dir: ./templates
target_dir: ./output

vars:
  env: production
  instance_type: t3.large

flags:
  enable_monitoring: true
  attach_ebs: false

files:
  - path: main.tf
    changes:
      - type: set_attr
        block: resource.aws_instance.web
        attr: instance_type
        value: "{{ .Vars.instance_type }}"

      - type: set_attr
        block: resource.aws_instance.web
        attr: monitoring
        value: "true"
        if: "{{ .Flags.enable_monitoring }}"

      - type: remove_attr
        block: resource.aws_s3_bucket.assets
        attr: acl

      - type: add_block
        from_file: snippets/ebs_volume.tf
        if: "{{ .Flags.attach_ebs }}"
```

**3. Validate, preview, apply**

```bash
hclforge validate transform.yaml
hclforge apply transform.yaml --dry-run
hclforge apply transform.yaml
```

---

## CLI reference

### `hclforge apply <spec>`

Applies the transformation spec and writes output files.

| Flag | Default | Description |
|------|---------|-------------|
| `--dry-run` | false | Preview changes; do not write any files |
| `--source <dir>` | spec value | Override `source_dir` |
| `--target <dir>` | spec value | Override `target_dir` |
| `--var key=value` | — | Override a var (repeatable) |
| `--flag key=true\|false` | — | Override a flag (repeatable) |

```bash
hclforge apply transform.yaml
hclforge apply transform.yaml --dry-run
hclforge apply transform.yaml --var env=staging --var region=eu-west-1
hclforge apply transform.yaml --flag enable_monitoring=false
hclforge apply transform.yaml --source ./templates --target ./output/prod
```

### `hclforge validate <spec>`

Parses and validates the spec without touching any files. Checks for missing required fields and unknown change types.

```bash
hclforge validate transform.yaml
# ✓ spec is valid
```

---

## Spec reference

### Top level

```yaml
source_dir: ./templates   # default: ./templates
target_dir: ./output      # default: ./output

vars:
  key: value              # string values, available as {{ .Vars.key }}

flags:
  key: true               # boolean values, available as {{ .Flags.key }}

files:
  - path: main.tf         # relative to source_dir
    changes: [...]
```

### Block notation

Blocks are identified using dot notation:

| Terraform construct | Notation |
|---------------------|----------|
| `resource "aws_instance" "web"` | `resource.aws_instance.web` |
| `data "aws_ami" "ubuntu"` | `data.aws_ami.ubuntu` |
| `provider "aws"` | `provider.aws` |
| `variable "env"` | `variable.env` |
| `output "vpc_id"` | `output.vpc_id` |
| `module "vpc"` | `module.vpc` |
| `locals` | `locals` |

### Change types

#### `set_attr` — set an attribute value

```yaml
- type: set_attr
  block: resource.aws_instance.web
  attr: instance_type
  value: "{{ .Vars.instance_type }}"   # template expression
  if: "{{ .Flags.resize }}"            # optional condition
```

Values that start with `var.`, `local.`, `module.`, or `data.` are written as raw HCL references rather than string literals.

```yaml
- type: set_attr
  block: resource.aws_instance.web
  attr: ami
  value: var.ami_id                     # written as HCL reference, not "var.ami_id"
```

#### `remove_attr` — remove an attribute

```yaml
- type: remove_attr
  block: resource.aws_s3_bucket.assets
  attr: acl
  if: "{{ .Flags.remove_acl }}"
```

#### `remove_block` — remove an entire block

```yaml
- type: remove_block
  block: resource.aws_spot_instance_request.web_spot
  if: "{{ not .Flags.use_spot }}"
```

#### `copy_block` — copy a block from another file

```yaml
- type: copy_block
  block: resource.aws_iam_role.default
  from_file: ../shared/iam.tf           # relative to source_dir
  rename: app_role                      # optional: rename last label
  if: "{{ .Flags.use_custom_iam }}"
  with:                                 # optional: apply changes to the copy
    - type: set_attr
      attr: name
      value: "{{ .Vars.env }}-app-role"
```

#### `add_block` — inject a block from a snippet file

```yaml
- type: add_block
  from_file: snippets/ebs_volume.tf    # relative to source_dir
  if: "{{ .Flags.attach_ebs }}"
```

### Template expressions

Conditions and values support Go `text/template` syntax:

```yaml
if: "{{ .Flags.enable_monitoring }}"         # flag is true
if: "{{ not .Flags.use_spot }}"              # flag is false
if: "{{ and .Flags.x (not .Flags.y) }}"     # compound
if: "{{ eq .Vars.env \"production\" }}"      # string comparison
if: "{{ ne .Vars.env \"staging\" }}"         # string inequality

value: "{{ .Vars.env }}-bucket"              # interpolation in value
value: "{{ .Vars.env }}"                     # plain var substitution
```

---

## Development

```bash
# Install dependencies
go mod tidy

# Run all tests
make test

# Run tests with coverage report
make coverage

# Build binary
make build

# Format and vet
make fmt vet

# Lint (requires golangci-lint)
make lint
```

### Project structure

```
hclforge/
├── cmd/hclforge/         # CLI entrypoint (cobra)
├── internal/
│   ├── config/           # YAML spec schema, loader, validator
│   ├── engine/           # orchestrates file processing and dry-run
│   ├── manipulator/      # hclwrite wrapper — low-level HCL read/write
│   ├── ops/              # one Apply function per change type
│   └── template/         # text/template evaluator for vars/flags
├── examples/
│   ├── templates/        # sample .tf template files
│   ├── snippets/         # sample reusable blocks
│   └── transform.yaml    # full working example spec
├── Makefile
└── .golangci.yml
```

---

## License

MIT — see [LICENSE](LICENSE).