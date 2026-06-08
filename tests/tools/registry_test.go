package tools_test

import (
	"strings"
	"testing"

	"vietclaw/internal/config"
	"vietclaw/internal/tools"
)

func TestToolRegistryDefinitionsRespectPolicy(t *testing.T) {
	cfg := config.Default(config.Paths{DataDir: t.TempDir()})
	cfg.Tools.Files.Enabled = true
	cfg.Tools.Shell.Enabled = false

	registry := tools.NewRegistry(cfg)
	defs := registry.GetDefinitions()

	foundRead := false
	foundWrite := false
	foundShell := false
	for _, d := range defs {
		switch d.Function.Name {
		case "file_read":
			foundRead = true
		case "file_write":
			foundWrite = true
		case "shell_exec":
			foundShell = true
		}
	}

	if !foundRead || !foundWrite {
		t.Errorf("expected file_read and file_write to be present, defs: %v", defs)
	}
	if foundShell {
		t.Errorf("shell_exec should not be present when disabled")
	}

	cfg.Tools.Shell.Enabled = true
	registry = tools.NewRegistry(cfg)
	defs = registry.GetDefinitions()
	foundShell = false
	for _, d := range defs {
		if d.Function.Name == "shell_exec" {
			foundShell = true
		}
	}
	if !foundShell {
		t.Errorf("expected shell_exec to be present when enabled")
	}
}

func TestToolRegistryDefinitionsUseEnglish(t *testing.T) {
	cfg := config.Default(config.Paths{DataDir: t.TempDir()})
	cfg.Agent.Language = "en"
	registry := tools.NewRegistry(cfg)
	defs := registry.GetDefinitions()
	if len(defs) == 0 || !strings.Contains(defs[0].Function.Description, "Read a file") {
		t.Fatalf("expected english tool description, defs: %#v", defs)
	}
}
