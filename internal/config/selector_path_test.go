package config

import "testing"

func TestSelectorFromPath_ResourceNested(t *testing.T) {
	resolved, err := selectorFromPath("resource.google_service_account.nodes")
	if err != nil {
		t.Fatalf("selectorFromPath failed: %v", err)
	}

	if resolved.Type != "resource" {
		t.Fatalf("unexpected target type: %q", resolved.Type)
	}
	if len(resolved.Labels) != 2 || resolved.Labels[0] != "google_service_account" || resolved.Labels[1] != "nodes" {
		t.Fatalf("unexpected target labels: %+v", resolved.Labels)
	}
	if len(resolved.Parents) != 0 {
		t.Fatalf("expected no parents, got %+v", resolved.Parents)
	}
}

func TestSelectorFromPath_DeepNested(t *testing.T) {
	resolved, err := selectorFromPath("resource.google_container_node_pool.pool.node_config.shielded_instance_config")
	if err != nil {
		t.Fatalf("selectorFromPath failed: %v", err)
	}

	if resolved.Type != "shielded_instance_config" {
		t.Fatalf("unexpected target type: %q", resolved.Type)
	}

	if len(resolved.Parents) != 2 {
		t.Fatalf("expected 2 parents, got %d", len(resolved.Parents))
	}

	if resolved.Parents[0].SelectedType() != "resource" {
		t.Fatalf("unexpected parent[0] type: %q", resolved.Parents[0].SelectedType())
	}
	if len(resolved.Parents[0].Labels) != 2 || resolved.Parents[0].Labels[0] != "google_container_node_pool" || resolved.Parents[0].Labels[1] != "pool" {
		t.Fatalf("unexpected parent[0] labels: %+v", resolved.Parents[0].Labels)
	}
	if resolved.Parents[1].SelectedType() != "node_config" {
		t.Fatalf("unexpected parent[1] type: %q", resolved.Parents[1].SelectedType())
	}
}

func TestSelectorFromPath_Invalid(t *testing.T) {
	_, err := selectorFromPath("resource.google_service_account")
	if err == nil {
		t.Fatalf("expected error for invalid path")
	}
}

func TestSelectorFromPath_Module(t *testing.T) {
	resolved, err := selectorFromPath("module.gke_cluster")
	if err != nil {
		t.Fatalf("selectorFromPath failed: %v", err)
	}

	if resolved.Type != "module" {
		t.Fatalf("unexpected target type: %q", resolved.Type)
	}
	if len(resolved.Labels) != 1 || resolved.Labels[0] != "gke_cluster" {
		t.Fatalf("unexpected target labels: %+v", resolved.Labels)
	}
}

func TestSelectorFromPath_Provider(t *testing.T) {
	resolved, err := selectorFromPath("provider.google")
	if err != nil {
		t.Fatalf("selectorFromPath failed: %v", err)
	}

	if resolved.Type != "provider" {
		t.Fatalf("unexpected target type: %q", resolved.Type)
	}
	if len(resolved.Labels) != 1 || resolved.Labels[0] != "google" {
		t.Fatalf("unexpected target labels: %+v", resolved.Labels)
	}
}

func TestSelectorFromPath_LocalsNested(t *testing.T) {
	resolved, err := selectorFromPath("locals.my_nested_block")
	if err != nil {
		t.Fatalf("selectorFromPath failed: %v", err)
	}

	if resolved.Type != "my_nested_block" {
		t.Fatalf("unexpected target type: %q", resolved.Type)
	}

	if len(resolved.Parents) != 1 {
		t.Fatalf("expected 1 parent, got %d", len(resolved.Parents))
	}

	if resolved.Parents[0].SelectedType() != "locals" {
		t.Fatalf("unexpected parent type: %q", resolved.Parents[0].SelectedType())
	}
}

func TestSelectorFromPath_RejectsEmptySegment(t *testing.T) {
	_, err := selectorFromPath("resource..google_service_account.nodes")
	if err == nil {
		t.Fatalf("expected error for empty segment")
	}
}

func TestSelectorFromPath_ModuleWithTwoLabels(t *testing.T) {
	resolved, err := selectorFromPath("module.tfe_workspace.example3")
	if err != nil {
		t.Fatalf("selectorFromPath failed: %v", err)
	}

	if resolved.Type != "module" {
		t.Fatalf("unexpected target type: %q", resolved.Type)
	}
	if len(resolved.Labels) != 2 || resolved.Labels[0] != "tfe_workspace" || resolved.Labels[1] != "example3" {
		t.Fatalf("unexpected target labels: %+v", resolved.Labels)
	}
}

func TestSelectorFromPath_ModuleWithTwoLabelsNestedProvider(t *testing.T) {
	resolved, err := selectorFromPath("module.tfe_workspace.example3.provider.tfe")
	if err != nil {
		t.Fatalf("selectorFromPath failed: %v", err)
	}

	if resolved.Type != "provider" {
		t.Fatalf("unexpected target type: %q", resolved.Type)
	}
	if len(resolved.Labels) != 1 || resolved.Labels[0] != "tfe" {
		t.Fatalf("unexpected target labels: %+v", resolved.Labels)
	}

	if len(resolved.Parents) != 1 {
		t.Fatalf("expected 1 parent, got %d", len(resolved.Parents))
	}

	if resolved.Parents[0].SelectedType() != "module" {
		t.Fatalf("unexpected parent type: %q", resolved.Parents[0].SelectedType())
	}
	if len(resolved.Parents[0].Labels) != 2 || resolved.Parents[0].Labels[0] != "tfe_workspace" || resolved.Parents[0].Labels[1] != "example3" {
		t.Fatalf("unexpected parent labels: %+v", resolved.Parents[0].Labels)
	}
}
