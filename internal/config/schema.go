// Package config handles loading, parsing, and validating the
// hclforge transformation spec (transform.yaml).
package config

// Spec is the top-level structure of a transform.yaml file.
type Spec struct {
	// SourceDir is the directory containing template .tf files.
	// Defaults to ./templates if not set.
	SourceDir string `yaml:"source_dir"`

	// TargetDir is the directory where output .tf files are written.
	// Defaults to ./output if not set.
	TargetDir string `yaml:"target_dir"`

	// Vars holds string values available as {{ .Vars.key }} in
	// change values and conditions.
	Vars map[string]string `yaml:"vars"`

	// Flags holds boolean values available as {{ .Flags.key }} in
	// conditions and values.
	Flags map[string]bool `yaml:"flags"`

	// Files is the ordered list of source files to process.
	Files []FileSpec `yaml:"files"`
}

// FileSpec defines the path to a source .tf file and the list of
// changes to apply to it in order.
type FileSpec struct {
	// Path is the path to the source .tf file, relative to SourceDir.
	Path string `yaml:"path"`

	// Changes is the ordered list of transformations to apply to the file.
	Changes []Change `yaml:"changes"`
}

// Change represents a single transformation operation to apply to a .tf file.
// Type determines which fields are required.
type Change struct {
	// Type is the operation to perform. Valid values:
	// set_attr, remove_attr, remove_block, copy_block, add_block.
	Type string `yaml:"type"`

	// Block identifies the target block using dot notation.
	// For example: "resource.aws_instance.web", "provider.aws".
	Block string `yaml:"block"`

	// Attr is the attribute name used by set_attr and remove_attr.
	// For example: "instance_type", "bucket", "tags".
	Attr string `yaml:"attr"`

	// Value is the new attribute value used by set_attr.
	// Supports Go template expressions: {{ .Vars.x }}, {{ .Flags.x }}.
	Value string `yaml:"value"`

	// If is a condition that must evaluate to true for the change to apply.
	// An empty condition is treated as unconditionally true.
	If string `yaml:"if"`

	// Rename sets a new last label on the copied block in copy_block.
	// For example: rename "web" to "prod" in resource.aws_instance.web.
	Rename string `yaml:"rename"`

	// FromFile is the path to a .tf snippet file, relative to SourceDir.
	// Used by copy_block and add_block.
	FromFile string `yaml:"from_file"`

	// With is a list of changes applied to the copied block after copying.
	// Only used by copy_block.
	With []Change `yaml:"with"`
}

// BlockRef holds the parsed components of a dot-notation block identifier.
// For example: "resource.aws_instance.web" parses to
// BlockRef{Type: "resource", Labels: ["aws_instance", "web"]}.
type BlockRef struct {
	// Type is the HCL block type, e.g. "resource", "provider", "variable".
	Type string

	// Labels are the ordered block labels after the type.
	// For example: ["aws_instance", "web"] from "resource.aws_instance.web".
	Labels []string
}
