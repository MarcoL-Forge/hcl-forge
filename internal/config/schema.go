// Package config handles loading, parsing, and validating the hclforge
// transformation spec (transform.yaml).
package config

// Spec is the top-level structure of a transform.yaml file.
type Spec struct {
	// SourceDir is the directory containing template .tf files. Default: ./templates
	SourceDir string `yaml:"source_dir"`

	// TargetDir is the directory where output .tf files are written. Default: ./output
	TargetDir string `yaml:"target_dir"`

	// Vars are string values available as {{ .Vars.key }} in change values and conditions.
	Vars map[string]string `yaml:"vars"`

	// Flags are boolean values available as {{ .Flags.key }} in conditions and values.
	Flags map[string]bool `yaml:"flags"`

	// Files lists the source files to process and their associated changes.
	Files []FileSpec `yaml:"files"`
}

// FileSpec describes one source file and the ordered list of changes to apply to it.
type FileSpec struct {
	// Path is the file path relative to source_dir.
	Path string `yaml:"path"`

	// Changes is an ordered list of transformations to apply.
	Changes []Change `yaml:"changes"`
}

// Change represents a single transformation operation.
type Change struct {
	// Type is the operation to perform. One of:
	// set_attr, remove_attr, remove_block, copy_block, add_block.
	Type string `yaml:"type"`

	// Block identifies the target block using dot notation: "resource.aws_instance.web".
	Block string `yaml:"block"`

	// Attr is the attribute name used by set_attr and remove_attr.
	Attr string `yaml:"attr"`

	// Value is the new attribute value for set_attr.
	// Supports Go text/template expressions: {{ .Vars.x }}, {{ .Flags.x }}.
	Value string `yaml:"value"`

	// If is an optional condition expression. The change is skipped when false.
	// Supports Go text/template: "{{ .Flags.enable_monitoring }}", "{{ not .Flags.use_spot }}".
	If string `yaml:"if"`

	// Rename sets a new last label when used with copy_block.
	// e.g. rename: "prod" changes resource "aws_instance" "web" -> "prod".
	Rename string `yaml:"rename"`

	// FromFile is a path to a .tf snippet file used by copy_block and add_block.
	FromFile string `yaml:"from_file"`

	// With contains nested changes applied to a copied block (copy_block only).
	With []Change `yaml:"with"`
}

// BlockRef holds the parsed components of a dot-notation block identifier.
//
// Examples:
//
//	"resource.aws_instance.web" -> BlockRef{Type: "resource", Labels: ["aws_instance", "web"]}
//	"provider.aws"              -> BlockRef{Type: "provider",  Labels: ["aws"]}
//	"locals"                    -> BlockRef{Type: "locals",    Labels: []}
type BlockRef struct {
	// Type is the HCL block type, e.g. "resource", "data", "provider".
	Type string

	// Labels are the ordered block labels, e.g. ["aws_instance", "web"].
	Labels []string
}