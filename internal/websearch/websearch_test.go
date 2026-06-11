package websearch

import (
	"os"
	"path/filepath"
	"testing"

	"vietclaw/internal/config"
)

func TestEntryRelPathConstants(t *testing.T) {
	if EntryRelPath != "build/index.js" {
		t.Fatalf("unexpected entry rel path: %s", EntryRelPath)
	}
	if SubProjectDirName != "open-websearch" {
		t.Fatalf("unexpected sub project dir name: %s", SubProjectDirName)
	}
}

func TestLocatorEnvOverride(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VIETCLAW_OPEN_WEBSEARCH_DIR", dir)
	got, err := NewLocator("").Resolve()
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got != dir {
		t.Fatalf("got %s want %s", got, dir)
	}
}

func TestUpsertAndRemove(t *testing.T) {
	cfg := config.Config{}
	server := config.MCPServerConfig{ID: ServerID, Enabled: true, Command: "node", Args: []string{"x"}}
	cfg = UpsertMCPServer(cfg, server)
	if len(cfg.Tools.MCP) != 1 || cfg.Tools.MCP[0].ID != ServerID {
		t.Fatalf("upsert insert failed: %+v", cfg.Tools.MCP)
	}
	updated := server
	updated.TimeoutSeconds = 99
	cfg = UpsertMCPServer(cfg, updated)
	if len(cfg.Tools.MCP) != 1 || cfg.Tools.MCP[0].TimeoutSeconds != 99 {
		t.Fatalf("upsert update failed: %+v", cfg.Tools.MCP)
	}
	got, ok := Find(cfg)
	if !ok || got.TimeoutSeconds != 99 {
		t.Fatalf("find after upsert failed: ok=%v got=%+v", ok, got)
	}
	cfg, ok = SetEnabled(cfg, false)
	if !ok || cfg.Tools.MCP[0].Enabled {
		t.Fatalf("SetEnabled disable failed: ok=%v cfg=%+v", ok, cfg.Tools.MCP)
	}
	cfg = RemoveMCPServer(cfg)
	if len(cfg.Tools.MCP) != 0 {
		t.Fatalf("RemoveMCPServer failed: %+v", cfg.Tools.MCP)
	}
}

func TestToolNames(t *testing.T) {
	names := ToolNames()
	want := map[string]bool{
		"search":               true,
		"fetchLinuxDoArticle":  true,
		"fetchCsdnArticle":     true,
		"fetchGithubReadme":    true,
		"fetchJuejinArticle":   true,
		"fetchWebContent":      true,
	}
	if len(names) != len(want) {
		t.Fatalf("expected %d tool names, got %d (%v)", len(want), len(names), names)
	}
	for _, n := range names {
		if !want[n] {
			t.Fatalf("unexpected tool name %q", n)
		}
	}
}

func TestMCPServerConfigRequiresBuild(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := MCPServerConfig(dir, nil); err == nil {
		t.Fatal("expected error when build/index.js is missing")
	}
}
