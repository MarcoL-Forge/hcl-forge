package config

type Config struct {
	Version int          `yaml:"version"`
	Input   InputConfig  `yaml:"input"`
	Output  OutputConfig `yaml:"output"`
	Options Options      `yaml:"options"`
	Edits   []EditConfig `yaml:"edits"`
}

type InputConfig struct {
	RootDir string   `yaml:"root_dir"`
	Files   []string `yaml:"files"`
}

type OutputConfig struct {
	Mode      string `yaml:"mode"`
	TargetDir string `yaml:"target_dir"`
}

type Options struct {
	Workers        int  `yaml:"workers"`
	FailOnNoChange bool `yaml:"fail_on_no_change"`
}

type EditConfig struct {
	Type string `yaml:"type"`

	// search_replace
	Old string `yaml:"old"`
	New string `yaml:"new"`

	// insert_hcl
	HCL string `yaml:"hcl"`

	// delete_hcl
	DeleteAll bool `yaml:"delete_all"`

	// future HCL-aware edits
	Block     *BlockSelector `yaml:"block"`
	Attribute string         `yaml:"attribute"`
}

type BlockSelector struct {
	Type      string   `yaml:"type"`
	BlockType string   `yaml:"block_type"`
	Labels    []string `yaml:"labels"`
}

// SelectedType returns the effective selector type.
// block_type is preferred; type is kept for backward compatibility.
func (s BlockSelector) SelectedType() string {
	if s.BlockType != "" {
		return s.BlockType
	}

	return s.Type
}
