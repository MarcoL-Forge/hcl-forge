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
		{name: "output file_map valid", mutate: func(c *Config) {
			c.Output.Mode = "target_dir"
			c.Output.TargetDir = "./out"
			c.Output.FileMap = map[string]string{"main.tf": "renamed/path/output.tf"}
		}, wantErr: false},
		{name: "output file_map requires target_dir mode", mutate: func(c *Config) {
			c.Output.Mode = "overwrite"
			c.Output.FileMap = map[string]string{"main.tf": "renamed.tf"}
		}, wantErr: true},
		{name: "output file_map unknown input", mutate: func(c *Config) {
			c.Output.Mode = "target_dir"
			c.Output.TargetDir = "./out"
			c.Output.FileMap = map[string]string{"missing.tf": "renamed.tf"}
		}, wantErr: true},
		{name: "output file_map empty output", mutate: func(c *Config) {
			c.Output.Mode = "target_dir"
			c.Output.TargetDir = "./out"
			c.Output.FileMap = map[string]string{"main.tf": ""}
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
		{name: "insert_hcl path selector accepted", mutate: func(c *Config) {
			c.Edits = []EditConfig{{
				Type: "insert_hcl",
				HCL:  "force_destroy = true",
				Block: &BlockSelector{
					Path: "resource.google_service_account.nodes",
				},
			}}
		}, wantErr: false},
		{name: "insert_hcl path cannot mix with explicit selector", mutate: func(c *Config) {
			c.Edits = []EditConfig{{
				Type: "insert_hcl",
				HCL:  "force_destroy = true",
				Block: &BlockSelector{
					Path:      "resource.google_service_account.nodes",
					BlockType: "resource",
				},
			}}
		}, wantErr: true},
		{name: "insert_hcl ensure requires block", mutate: func(c *Config) {
			c.Edits = []EditConfig{{
				Type:              "insert_hcl",
				HCL:               "force_destroy = true",
				EnsureTargetBlock: true,
			}}
		}, wantErr: true},
		{name: "insert_hcl guard requires block", mutate: func(c *Config) {
			c.Edits = []EditConfig{{
				Type:  "insert_hcl",
				HCL:   "force_destroy = true",
				Guard: &GuardConfig{IfTargetExists: true},
			}}
		}, wantErr: true},
		{name: "insert_hcl conflicting guard conditions", mutate: func(c *Config) {
			c.Edits = []EditConfig{{
				Type: "insert_hcl",
				HCL:  "force_destroy = true",
				Guard: &GuardConfig{
					IfTargetExists:  true,
					IfTargetMissing: true,
				},
				Block: &BlockSelector{
					Path: "resource.google_service_account.nodes",
				},
			}}
		}, wantErr: true},
		{name: "insert_hcl guard valid", mutate: func(c *Config) {
			c.Edits = []EditConfig{{
				Type:              "insert_hcl",
				HCL:               "force_destroy = true",
				EnsureTargetBlock: true,
				Guard:             &GuardConfig{IfTargetMissing: true},
				Block: &BlockSelector{
					Path: "resource.google_service_account.nodes",
				},
			}}
		}, wantErr: false},
		{name: "insert_hcl parent missing type", mutate: func(c *Config) {
			c.Edits = []EditConfig{{
				Type: "insert_hcl",
				HCL:  "force_destroy = true",
				Block: &BlockSelector{
					BlockType: "node_config",
					Labels:    []string{},
					Parents:   []ParentSelector{{Type: "", BlockType: "", Labels: []string{"x"}}},
				},
			}}
		}, wantErr: true},
		{name: "delete_hcl with attribute", mutate: func(c *Config) {
			c.Edits = []EditConfig{{
				Type:      "delete_hcl",
				Attribute: "location",
			}}
		}, wantErr: false},
		{name: "delete_hcl with block", mutate: func(c *Config) {
			c.Edits = []EditConfig{{
				Type: "delete_hcl",
				Block: &BlockSelector{
					BlockType: "variable",
					Labels:    []string{"project_id"},
				},
			}}
		}, wantErr: false},
		{name: "delete_hcl missing selectors", mutate: func(c *Config) {
			c.Edits = []EditConfig{{Type: "delete_hcl"}}
		}, wantErr: true},
		{name: "delete_hcl delete_all with attribute", mutate: func(c *Config) {
			c.Edits = []EditConfig{{
				Type:      "delete_hcl",
				Attribute: "location",
				DeleteAll: true,
			}}
		}, wantErr: false},
		{name: "delete_hcl keep_only valid", mutate: func(c *Config) {
			c.Edits = []EditConfig{{
				Type:      "delete_hcl",
				KeepOnly:  true,
				MatchMode: "glob",
				Block: &BlockSelector{
					BlockType: "resource",
					Labels:    []string{"tfe_workspace", "example*"},
				},
			}}
		}, wantErr: false},
		{name: "delete_hcl keep_only requires block", mutate: func(c *Config) {
			c.Edits = []EditConfig{{
				Type:     "delete_hcl",
				KeepOnly: true,
			}}
		}, wantErr: true},
		{name: "delete_hcl keep_only cannot combine attribute", mutate: func(c *Config) {
			c.Edits = []EditConfig{{
				Type:      "delete_hcl",
				KeepOnly:  true,
				Attribute: "location",
				Block: &BlockSelector{
					BlockType: "resource",
					Labels:    []string{"google_storage_bucket", "bucket"},
				},
			}}
		}, wantErr: true},
		{name: "delete_hcl invalid match mode", mutate: func(c *Config) {
			c.Edits = []EditConfig{{
				Type:      "delete_hcl",
				Attribute: "location",
				MatchMode: "invalid",
			}}
		}, wantErr: true},
		{name: "delete_hcl regex match mode", mutate: func(c *Config) {
			c.Edits = []EditConfig{{
				Type:      "delete_hcl",
				DeleteAll: true,
				MatchMode: "regex",
				Block: &BlockSelector{
					BlockType: "module",
					Labels:    []string{"service-account-(dev|prod)"},
				},
			}}
		}, wantErr: false},
		{name: "delete_hcl path selector accepted", mutate: func(c *Config) {
			c.Edits = []EditConfig{{
				Type: "delete_hcl",
				Block: &BlockSelector{
					Path: "resource.google_service_account.nodes",
				},
			}}
		}, wantErr: false},
		{name: "delete_hcl parent missing type", mutate: func(c *Config) {
			c.Edits = []EditConfig{{
				Type: "delete_hcl",
				Block: &BlockSelector{
					BlockType: "variable",
					Labels:    []string{"project_id"},
					Parents:   []ParentSelector{{Type: "", BlockType: "", Labels: []string{"x"}}},
				},
			}}
		}, wantErr: true},
		{name: "set_attribute valid", mutate: func(c *Config) {
			c.Edits = []EditConfig{{
				Type:            "set_attribute",
				Attribute:       "force_destroy",
				ValueHCL:        "true",
				CreateIfMissing: true,
				Block: &BlockSelector{
					Path: "resource.google_storage_bucket.bucket",
				},
			}}
		}, wantErr: false},
		{name: "set_attribute missing attribute", mutate: func(c *Config) {
			c.Edits = []EditConfig{{
				Type:     "set_attribute",
				ValueHCL: "true",
			}}
		}, wantErr: true},
		{name: "set_attribute missing value_hcl", mutate: func(c *Config) {
			c.Edits = []EditConfig{{
				Type:      "set_attribute",
				Attribute: "force_destroy",
			}}
		}, wantErr: true},
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
