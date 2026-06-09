package tools

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
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
	if err := t.Policy.ShellNetworkAllowed(input); err != nil {
		return "", err
	}
	if t.Policy.cfg.Tools.Shell.Sandbox == "docker" {
		return t.runDocker(ctx, fields)
	}
	if runtime.GOOS == "windows" {
		out, err := exec.CommandContext(ctx, "cmd", "/C", input).CombinedOutput()
		return string(out), err
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

var shellURLPattern = regexp.MustCompile(`(?i)\bhttps?://[^\s"'<>]+`)

func (p Policy) ShellNetworkAllowed(command string) error {
	policy := p.cfg.Tools.Shell.NetworkPolicy
	if !policy.Enabled {
		return nil
	}
	for _, rawURL := range shellURLPattern.FindAllString(command, -1) {
		parsed, err := url.Parse(rawURL)
		if err != nil {
			return fmt.Errorf("blocked shell network target: invalid URL %s", rawURL)
		}
		if err := p.hostAllowed(parsed.Hostname()); err != nil {
			return err
		}
	}
	for _, field := range strings.Fields(command) {
		candidate := strings.Trim(field, `"'[](),;`)
		if strings.Contains(candidate, "/") || strings.Contains(candidate, "\\") || strings.Contains(candidate, "$") {
			continue
		}
		if ip := net.ParseIP(candidate); ip != nil {
			if err := p.ipAllowed(candidate, ip); err != nil {
				return err
			}
		}
	}
	return nil
}

func (p Policy) HTTPURLAllowed(rawURL string) error {
	policy := p.cfg.Tools.Shell.NetworkPolicy
	if !policy.Enabled {
		return nil
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("blocked network target: invalid URL %s", rawURL)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("blocked network target: unsupported scheme %s", parsed.Scheme)
	}
	return p.hostAllowed(parsed.Hostname())
}

func (p Policy) hostAllowed(host string) error {
	host = strings.Trim(strings.ToLower(host), ".")
	if host == "" {
		return nil
	}
	policy := p.cfg.Tools.Shell.NetworkPolicy
	for _, pattern := range policy.DenyHosts {
		if hostMatches(host, pattern) {
			return fmt.Errorf("blocked shell network host: %s", host)
		}
	}
	if ip := net.ParseIP(host); ip != nil {
		return p.ipAllowed(host, ip)
	}
	if policy.DenyPrivate {
		addrs, err := net.LookupIP(host)
		if err == nil {
			for _, ip := range addrs {
				if privateIP(ip) {
					return fmt.Errorf("blocked shell network host: %s resolves to private IP", host)
				}
			}
		}
	}
	if policy.RestrictToAllowHosts && !hostInPatterns(host, policy.AllowHosts) {
		return fmt.Errorf("blocked shell network host: %s is not in allow list", host)
	}
	return nil
}

func (p Policy) ipAllowed(label string, ip net.IP) error {
	if p.cfg.Tools.Shell.NetworkPolicy.DenyPrivate && privateIP(ip) {
		return fmt.Errorf("blocked shell network IP: %s", label)
	}
	return nil
}

func hostMatches(host string, pattern string) bool {
	pattern = strings.Trim(strings.ToLower(pattern), ".")
	if pattern == "" {
		return false
	}
	if strings.HasPrefix(pattern, "*.") {
		suffix := strings.TrimPrefix(pattern, "*")
		return strings.HasSuffix(host, suffix)
	}
	return host == pattern
}

func hostInPatterns(host string, patterns []string) bool {
	for _, pattern := range patterns {
		if hostMatches(host, pattern) {
			return true
		}
	}
	return false
}

func privateIP(ip net.IP) bool {
	return ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified()
}
