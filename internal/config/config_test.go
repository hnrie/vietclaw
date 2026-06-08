package config

import (
	"path/filepath"
	"testing"
)

func TestDefaultIncludesAgentRuntime(t *testing.T) {
	dir := t.TempDir()
	cfg := Default(Paths{DataDir: dir})

	if cfg.Agent.Name != "VietClaw" {
		t.Fatalf("agent name = %q", cfg.Agent.Name)
	}
	if cfg.Agent.Workspace != filepath.Join(dir, "workspace") {
		t.Fatalf("workspace = %q", cfg.Agent.Workspace)
	}
	if len(cfg.Providers) != 1 || cfg.Providers[0].ID != "mock" || !cfg.Providers[0].Enabled {
		t.Fatalf("default mock provider missing: %#v", cfg.Providers)
	}
	if cfg.Router.DefaultProvider != "mock" || cfg.Router.DefaultModel != "mock-small" {
		t.Fatalf("router default invalid: %#v", cfg.Router)
	}
	if cfg.Tools.Shell.Enabled {
		t.Fatal("shell must be disabled by default")
	}
	if !cfg.Tools.Files.Enabled || !cfg.Tools.Files.WorkspaceOnly {
		t.Fatalf("file tools default invalid: %#v", cfg.Tools.Files)
	}
}

func TestMergeDefaultKeepsExistingValues(t *testing.T) {
	def := Default(Paths{DataDir: t.TempDir()})
	cfg := Config{}
	cfg.Server.Host = "0.0.0.0"

	merged := MergeDefault(cfg, def)
	if merged.Server.Host != "0.0.0.0" {
		t.Fatalf("existing host was overwritten: %s", merged.Server.Host)
	}
	if merged.Agent.MaxContextChars == 0 || len(merged.Providers) == 0 {
		t.Fatalf("defaults were not merged: %#v", merged)
	}
}
