package playbook

type Playbook struct {
	Version    int         `yaml:"version"`
	Input      string      `yaml:"input"`
	Output     string      `yaml:"output"`
	Operations []Operation `yaml:"operations"`
}

type Operation struct {
	Op        string   `yaml:"op"`
	Target    string   `yaml:"target"`
	BlockType string   `yaml:"block_type"`
	Labels    []string `yaml:"labels"`
	Attribute string   `yaml:"attribute"`
	Value     string   `yaml:"value"`
	ValueType string   `yaml:"value_type"`
}
