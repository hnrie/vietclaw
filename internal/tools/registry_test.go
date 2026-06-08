package tools

import (
	"testing"
	"vietclaw/internal/config"
)

func TestToolRegistry(t *testing.T) {
	cfg := config.Config{}
	cfg.Tools.Files.Enabled = true
	cfg.Tools.Shell.Enabled = false

	registry := NewRegistry(cfg)
	defs := registry.GetDefinitions()

	foundRead := false
	foundWrite := false
	foundShell := false

	for _, d := range defs {
		if d.Function.Name == "file_read" {
			foundRead = true
		}
		if d.Function.Name == "file_write" {
			foundWrite = true
		}
		if d.Function.Name == "shell_exec" {
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
	registry2 := NewRegistry(cfg)
	defs2 := registry2.GetDefinitions()
	foundShell2 := false
	for _, d := range defs2 {
		if d.Function.Name == "shell_exec" {
			foundShell2 = true
		}
	}
	if !foundShell2 {
		t.Errorf("expected shell_exec to be present when enabled")
	}
}
