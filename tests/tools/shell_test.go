package tools_test

import (
	"path/filepath"
	"reflect"
	"testing"

	"vietclaw/internal/config"
	"vietclaw/internal/tools"
)

func TestBuildDockerShellArgs(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Default(config.Paths{DataDir: dir})
	cfg.Tools.Shell.Sandbox = "docker"
	cfg.Tools.Shell.DockerImage = "busybox:1.36"
	cfg.Tools.Shell.DockerNetwork = "none"
	cfg.Tools.Shell.WorkspaceMode = "ro"

	workspace, err := filepath.Abs(cfg.Agent.Workspace)
	if err != nil {
		t.Fatal(err)
	}
	got := tools.BuildDockerShellArgs(cfg, []string{"echo", "hello"})
	want := []string{
		"run",
		"--rm",
		"--network", "none",
		"-v", workspace + ":/workspace:ro",
		"-w", "/workspace",
		"busybox:1.36",
		"echo", "hello",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("docker args = %#v, want %#v", got, want)
	}
}
