package tools

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"vietclaw/internal/config"
)

type ShellExec struct {
	Policy Policy
}

func (t ShellExec) Name() string { return "shell.exec" }

func (t ShellExec) Run(ctx context.Context, input string) (string, error) {
	if !t.Policy.ShellAllowed() {
		return "", fmt.Errorf("shell.exec disabled")
	}
	fields := strings.Fields(input)
	if len(fields) == 0 {
		return "", fmt.Errorf("empty command")
	}
	if t.Policy.cfg.Tools.Shell.Sandbox == "docker" {
		return t.runDocker(ctx, fields)
	}
	out, err := exec.CommandContext(ctx, fields[0], fields[1:]...).CombinedOutput()
	return string(out), err
}

func (t ShellExec) runDocker(ctx context.Context, command []string) (string, error) {
	cfg := t.Policy.cfg.Tools.Shell
	workspace, err := filepath.Abs(config.ExpandPath(t.Policy.cfg.Agent.Workspace))
	if err != nil {
		return "", err
	}
	runCtx := ctx
	cancel := func() {}
	if cfg.TimeoutSeconds > 0 {
		runCtx, cancel = context.WithTimeout(ctx, time.Duration(cfg.TimeoutSeconds)*time.Second)
	}
	defer cancel()

	args := BuildDockerShellArgs(t.Policy.cfg, command)
	out, err := exec.CommandContext(runCtx, defaultString(cfg.DockerBinary, config.DefaultDockerBinary), args...).CombinedOutput()
	if runCtx.Err() == context.DeadlineExceeded {
		return string(out), fmt.Errorf("shell.exec docker timeout")
	}
	if !strings.Contains(strings.Join(args, " "), workspace) {
		return string(out), fmt.Errorf("shell.exec docker workspace mount missing")
	}
	return string(out), err
}

func BuildDockerShellArgs(cfg config.Config, command []string) []string {
	shellCfg := cfg.Tools.Shell
	workspace, _ := filepath.Abs(config.ExpandPath(cfg.Agent.Workspace))
	mode := defaultString(shellCfg.WorkspaceMode, config.DefaultWorkspaceMode)
	if mode != "rw" {
		mode = "ro"
	}
	network := defaultString(shellCfg.DockerNetwork, config.DefaultDockerNetwork)
	image := defaultString(shellCfg.DockerImage, config.DefaultDockerImage)

	args := []string{
		"run",
		"--rm",
		"--network", network,
		"-v", workspace + ":/workspace:" + mode,
		"-w", "/workspace",
		image,
	}
	return append(args, command...)
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
