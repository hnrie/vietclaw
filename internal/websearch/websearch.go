// Package websearch integrates the bundled open-websearch MCP sub project
// (cloned from https://github.com/aas-ee/open-websearch) into VietClaw.
//
// The sub project lives at <repo>/open-websearch and exposes a Node-based
// Model Context Protocol server with web search and content fetch tools.
// This package owns:
//
//   - locating the sub project on disk
//   - building it (npm install + npm run build)
//   - producing a config.MCPServerConfig that VietClaw's tool registry can
//     spawn over stdio so its tools (search, fetchWebContent, fetchCsdnArticle,
//     fetchGithubReadme, fetchJuejinArticle, fetchLinuxDoArticle, ...) become
//     usable inside VietClaw agents.
package websearch

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"vietclaw/internal/config"
)

// ServerID is the canonical id used inside VietClaw's MCP config for the
// open-websearch sub project. The tool registry will prefix discovered tool
// names with `mcp_<server_id>_<tool>`.
const ServerID = "open_websearch"

// SubProjectDirName is the directory name of the bundled sub project inside
// the VietClaw repository.
const SubProjectDirName = "open-websearch"

// EntryRelPath is the relative path of the built Node entrypoint emitted by
// `npm run build` inside the sub project.
const EntryRelPath = "build/index.js"

// DefaultInstallTimeout caps `npm install` runs triggered by VietClaw.
const DefaultInstallTimeout = 5 * time.Minute

// DefaultBuildTimeout caps `npm run build` runs triggered by VietClaw.
const DefaultBuildTimeout = 5 * time.Minute

// DefaultMCPTimeoutSeconds bounds a single stdio MCP call. Web searches can
// be slow on first cold engines, so the default is generous.
const DefaultMCPTimeoutSeconds = 45

// Locator resolves the on-disk location of the sub project. It searches a
// chain of candidates: VIETCLAW_OPEN_WEBSEARCH_DIR env override, the current
// working directory, the VietClaw repo root (relative to the caller), and the
// configured data dir.
type Locator struct {
	DataDir string
}

// NewLocator returns a Locator that resolves the sub project against the
// running process and the provided data dir.
func NewLocator(dataDir string) Locator {
	return Locator{DataDir: dataDir}
}

// Resolve finds the sub project directory. It does not require the build
// directory to exist yet — that is the job of EnsureBuilt.
func (l Locator) Resolve() (string, error) {
	for _, candidate := range l.candidates() {
		if candidate == "" {
			continue
		}
		info, err := os.Stat(filepath.Join(candidate, "package.json"))
		if err == nil && !info.IsDir() {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("open-websearch sub project not found; expected %s/ inside the VietClaw checkout or VIETCLAW_OPEN_WEBSEARCH_DIR override", SubProjectDirName)
}

func (l Locator) candidates() []string {
	out := []string{}
	if env := strings.TrimSpace(os.Getenv("VIETCLAW_OPEN_WEBSEARCH_DIR")); env != "" {
		out = append(out, env)
	}
	if wd, err := os.Getwd(); err == nil {
		out = append(out, filepath.Join(wd, SubProjectDirName))
		out = append(out, filepath.Join(wd, "..", SubProjectDirName))
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		out = append(out, filepath.Join(dir, SubProjectDirName))
		out = append(out, filepath.Join(dir, "..", SubProjectDirName))
	}
	if l.DataDir != "" {
		out = append(out, filepath.Join(l.DataDir, SubProjectDirName))
	}
	return out
}

// EntryPath returns the absolute path of the Node entrypoint, or an error if
// the sub project has not been built yet.
func EntryPath(subProjectDir string) (string, error) {
	if subProjectDir == "" {
		return "", errors.New("sub project dir is empty")
	}
	entry := filepath.Join(subProjectDir, EntryRelPath)
	info, err := os.Stat(entry)
	if err != nil {
		return "", fmt.Errorf("open-websearch entry %s not found; run `vietclaw websearch build`: %w", entry, err)
	}
	if info.IsDir() {
		return "", fmt.Errorf("open-websearch entry %s is a directory", entry)
	}
	return entry, nil
}

// IsBuilt returns true when the bundled entrypoint exists on disk.
func IsBuilt(subProjectDir string) bool {
	if subProjectDir == "" {
		return false
	}
	_, err := os.Stat(filepath.Join(subProjectDir, EntryRelPath))
	return err == nil
}

// EnsureNodeAvailable returns the resolved node binary path or an error. The
// open-websearch MCP server is a Node app and must run under a real Node
// runtime; we do not vendor a JS interpreter.
func EnsureNodeAvailable() (string, error) {
	bin, err := exec.LookPath("node")
	if err != nil {
		return "", errors.New("node binary not found on PATH; install Node.js 18+ to use the open-websearch MCP server")
	}
	return bin, nil
}

// EnsureNPMAvailable returns the resolved npm binary path or an error.
func EnsureNPMAvailable() (string, error) {
	bin, err := exec.LookPath("npm")
	if err != nil {
		return "", errors.New("npm binary not found on PATH; install Node.js / npm to build the open-websearch MCP server")
	}
	return bin, nil
}

// Install runs `npm install` inside the sub project, streaming logs to the
// given writer.
func Install(ctx context.Context, subProjectDir string, logTo *os.File) error {
	npm, err := EnsureNPMAvailable()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, DefaultInstallTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, npm, "install", "--no-audit", "--no-fund")
	cmd.Dir = subProjectDir
	cmd.Env = append(os.Environ(), "CI=1")
	if logTo != nil {
		cmd.Stdout = logTo
		cmd.Stderr = logTo
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("npm install failed in %s: %w", subProjectDir, err)
	}
	return nil
}

// Build runs `npm run build` inside the sub project.
func Build(ctx context.Context, subProjectDir string, logTo *os.File) error {
	npm, err := EnsureNPMAvailable()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, DefaultBuildTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, npm, "run", "build")
	cmd.Dir = subProjectDir
	cmd.Env = os.Environ()
	if logTo != nil {
		cmd.Stdout = logTo
		cmd.Stderr = logTo
	}
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("npm run build failed in %s: %w", subProjectDir, err)
	}
	if !IsBuilt(subProjectDir) {
		return fmt.Errorf("npm run build completed but %s was not produced", EntryRelPath)
	}
	return nil
}

// EnsureBuilt installs (when node_modules is missing) and builds (when the
// entrypoint is missing) the sub project. Idempotent and safe to call on
// every daemon start when websearch is enabled.
func EnsureBuilt(ctx context.Context, subProjectDir string, logTo *os.File) error {
	if subProjectDir == "" {
		return errors.New("sub project dir is empty")
	}
	if _, err := os.Stat(subProjectDir); err != nil {
		return fmt.Errorf("sub project dir %s: %w", subProjectDir, err)
	}
	nm := filepath.Join(subProjectDir, "node_modules")
	if _, err := os.Stat(nm); err != nil {
		if err := Install(ctx, subProjectDir, logTo); err != nil {
			return err
		}
	}
	if IsBuilt(subProjectDir) {
		return nil
	}
	return Build(ctx, subProjectDir, logTo)
}

// MCPServerConfig produces the config.MCPServerConfig entry that VietClaw's
// MCP client will spawn over stdio. The sub project's `node build/index.js`
// entrypoint speaks MCP on stdin/stdout when MODE=stdio is set, and exposes
// search + fetch tools registered in src/tools/setupTools.ts.
func MCPServerConfig(subProjectDir string, env map[string]string) (config.MCPServerConfig, error) {
	node, err := EnsureNodeAvailable()
	if err != nil {
		return config.MCPServerConfig{}, err
	}
	entry, err := EntryPath(subProjectDir)
	if err != nil {
		return config.MCPServerConfig{}, err
	}
	mergedEnv := defaultEnv()
	for k, v := range env {
		mergedEnv[k] = v
	}
	return config.MCPServerConfig{
		ID:             ServerID,
		Enabled:        true,
		Transport:      "stdio",
		Command:        node,
		Args:           []string{entry},
		Env:            mergedEnv,
		TimeoutSeconds: DefaultMCPTimeoutSeconds,
	}, nil
}

func defaultEnv() map[string]string {
	env := map[string]string{
		// Force pure stdio so the server does not also try to bind an HTTP
		// port when launched as a VietClaw MCP child process.
		"MODE": "stdio",
		// Quiet npm-style noise into stderr where the MCP client ignores it.
		"NODE_NO_WARNINGS": "1",
	}
	if runtime.GOOS == "windows" {
		env["NPM_CONFIG_LOGLEVEL"] = "error"
	}
	return env
}

// ToolNames returns the canonical MCP tool ids exposed by the open-websearch
// server. These match the registrations in src/tools/setupTools.ts and are
// what the agent will see after the MCP client prefixes them with
// `mcp_<ServerID>_`.
func ToolNames() []string {
	return []string{
		"search",
		"fetchLinuxDoArticle",
		"fetchCsdnArticle",
		"fetchGithubReadme",
		"fetchJuejinArticle",
		"fetchWebContent",
	}
}
