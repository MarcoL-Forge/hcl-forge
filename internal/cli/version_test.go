package cli

import (
	"runtime/debug"
	"testing"
)

func TestResolvedVersion_UsesExplicitVersion(t *testing.T) {
	originalVersion := Version
	originalReadBuildInfo := readBuildInfo
	t.Cleanup(func() {
		Version = originalVersion
		readBuildInfo = originalReadBuildInfo
	})

	Version = "1.2.3"
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{Main: debug.Module{Version: "v9.9.9"}}, true
	}

	if got := resolvedVersion(); got != "1.2.3" {
		t.Fatalf("expected explicit version, got %q", got)
	}
}

func TestResolvedVersion_UsesBuildInfoForDev(t *testing.T) {
	originalVersion := Version
	originalReadBuildInfo := readBuildInfo
	t.Cleanup(func() {
		Version = originalVersion
		readBuildInfo = originalReadBuildInfo
	})

	Version = "dev"
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{Main: debug.Module{Version: "v0.9.1"}}, true
	}

	if got := resolvedVersion(); got != "v0.9.1" {
		t.Fatalf("expected build info version, got %q", got)
	}
}

func TestResolvedVersion_FallsBackToDevForDevelBuild(t *testing.T) {
	originalVersion := Version
	originalReadBuildInfo := readBuildInfo
	t.Cleanup(func() {
		Version = originalVersion
		readBuildInfo = originalReadBuildInfo
	})

	Version = "dev"
	readBuildInfo = func() (*debug.BuildInfo, bool) {
		return &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}}, true
	}

	if got := resolvedVersion(); got != "dev" {
		t.Fatalf("expected dev fallback, got %q", got)
	}
}
