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
	Workers         int  `yaml:"workers"`
	FailOnNoChange bool `yaml:"fail_on_no_change"`
}

type EditConfig struct {
	Type string `yaml:"type"`

	// search_replace
	Old string `yaml:"old"`
	New string `yaml:"new"`

	// future HCL-aware edits
	Block     *BlockSelector `yaml:"block"`
	Attribute string         `yaml:"attribute"`
}

type BlockSelector struct {
	Type   string   `yaml:"type"`
	Labels []string `yaml:"labels"`
}