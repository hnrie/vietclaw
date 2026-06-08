package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"vietclaw/internal/config"
)

type Tool interface {
	Name() string
	Run(ctx context.Context, input string) (string, error)
}

type Policy struct {
	cfg config.Config
}

func NewPolicy(cfg config.Config) Policy {
	return Policy{cfg: cfg}
}

func (p Policy) ShellAllowed() bool {
	return p.cfg.Tools.Shell.Enabled
}

func (p Policy) FileAllowed(path string) (string, error) {
	if !p.cfg.Tools.Files.Enabled {
		return "", fmt.Errorf("file tools disabled")
	}
	workspace := config.ExpandPath(p.cfg.Agent.Workspace)
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		return "", err
	}
	cleaned := filepath.Clean(path)
	if !filepath.IsAbs(cleaned) {
		cleaned = filepath.Join(workspace, cleaned)
	}
	abs, err := filepath.Abs(cleaned)
	if err != nil {
		return "", err
	}
	workspaceAbs, err := filepath.Abs(workspace)
	if err != nil {
		return "", err
	}
	if p.cfg.Tools.Files.WorkspaceOnly {
		rel, err := filepath.Rel(workspaceAbs, abs)
		if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return "", fmt.Errorf("path outside workspace")
		}
	}
	return abs, nil
}

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
	out, err := exec.CommandContext(ctx, fields[0], fields[1:]...).CombinedOutput()
	return string(out), err
}

type FileRead struct {
	Policy Policy
}

func (t FileRead) Name() string { return "file.read" }

func (t FileRead) Run(_ context.Context, input string) (string, error) {
	path, err := t.Policy.FileAllowed(input)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	return string(data), err
}

type FileWrite struct {
	Policy Policy
}

func (t FileWrite) Name() string { return "file.write" }

func (t FileWrite) Run(_ context.Context, input string) (string, error) {
	parts := strings.SplitN(input, "\n", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("file.write input must be path newline content")
	}
	path, err := t.Policy.FileAllowed(parts[0])
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, []byte(parts[1]), 0o644); err != nil {
		return "", err
	}
	return "ok", nil
}
