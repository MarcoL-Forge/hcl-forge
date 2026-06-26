package config

import "testing"

func TestApplyDefaults(t *testing.T) {
	cfg := Config{}
	cfg.ApplyDefaults()

	if cfg.Input.RootDir != "." {
		t.Fatalf("expected default root_dir '.', got %q", cfg.Input.RootDir)
	}
	if cfg.Options.Workers != 4 {
		t.Fatalf("expected default workers 4, got %d", cfg.Options.Workers)
	}
	if cfg.Output.Mode != "overwrite" {
		t.Fatalf("expected default output mode 'overwrite', got %q", cfg.Output.Mode)
	}
}

func TestValidate(t *testing.T) {
	base := Config{
		Version: 1,
		Input: InputConfig{
			RootDir: ".",
			Files:   []string{"main.tf"},
		},
		Output: OutputConfig{Mode: "overwrite"},
		Edits: []EditConfig{{
			Type: "search_replace",
			Old:  "old",
			New:  "new",
		}},
	}

	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr bool
	}{
		{name: "valid config", mutate: func(*Config) {}, wantErr: false},
		{name: "invalid version", mutate: func(c *Config) { c.Version = 2 }, wantErr: true},
		{name: "missing input files", mutate: func(c *Config) { c.Input.Files = nil }, wantErr: true},
		{name: "unsupported output mode", mutate: func(c *Config) { c.Output.Mode = "somewhere" }, wantErr: true},
		{name: "target dir mode missing target", mutate: func(c *Config) {
			c.Output.Mode = "target_dir"
			c.Output.TargetDir = ""
		}, wantErr: true},
		{name: "no edits", mutate: func(c *Config) { c.Edits = nil }, wantErr: true},
		{name: "missing edit type", mutate: func(c *Config) { c.Edits = []EditConfig{{Type: ""}} }, wantErr: true},
		{name: "search_replace missing old", mutate: func(c *Config) {
			c.Edits = []EditConfig{{Type: "search_replace", Old: "", New: "x"}}
		}, wantErr: true},
		{name: "unsupported edit type", mutate: func(c *Config) {
			c.Edits = []EditConfig{{Type: "unknown"}}
		}, wantErr: true},
		{name: "insert_hcl valid", mutate: func(c *Config) {
			c.Edits = []EditConfig{{Type: "insert_hcl", HCL: "variable \"x\" { type = string }"}}
		}, wantErr: false},
		{name: "insert_hcl missing snippet", mutate: func(c *Config) {
			c.Edits = []EditConfig{{Type: "insert_hcl"}}
		}, wantErr: true},
		{name: "insert_hcl block missing type", mutate: func(c *Config) {
			c.Edits = []EditConfig{{
				Type:  "insert_hcl",
				HCL:   "force_destroy = true",
				Block: &BlockSelector{Type: "", BlockType: "", Labels: []string{"a", "b"}},
			}}
		}, wantErr: true},
		{name: "insert_hcl block_type accepted", mutate: func(c *Config) {
			c.Edits = []EditConfig{{
				Type: "insert_hcl",
				HCL:  "force_destroy = true",
				Block: &BlockSelector{
					BlockType: "node_config",
					Labels:    []string{},
				},
			}}
		}, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := base
			cfg.Input.Files = append([]string(nil), base.Input.Files...)
			cfg.Edits = append([]EditConfig(nil), base.Edits...)

			tt.mutate(&cfg)

			err := Validate(cfg)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
		})
	}
}
