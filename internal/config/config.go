package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

type Paths struct {
	DataDir    string
	ConfigFile string
	LogDir     string
	LogFile    string
}

type Config struct {
	Server    ServerConfig     `json:"server"`
	Runtime   RuntimeConfig    `json:"runtime"`
	Database  DatabaseConfig   `json:"database"`
	Agent     AgentConfig      `json:"agent"`
	Channels  ChannelsConfig   `json:"channels"`
	Providers []ProviderConfig `json:"providers"`
	Router    RouterConfig     `json:"router"`
	Tools     ToolsConfig      `json:"tools"`
	Budget    BudgetConfig     `json:"budget"`
}

type ServerConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

type RuntimeConfig struct {
	Mode               string `json:"mode"`
	MaxConcurrentTasks int    `json:"max_concurrent_tasks"`
}

type DatabaseConfig struct {
	Path string `json:"path"`
}

type AgentConfig struct {
	Name               string `json:"name"`
	Language           string `json:"language"`
	Style              string `json:"style"`
	DefaultMode        string `json:"default_mode"`
	Workspace          string `json:"workspace"`
	MaxContextChars    int    `json:"max_context_chars"`
	MaxHistoryMessages int    `json:"max_history_messages"`
}

type ChannelsConfig struct {
	Discord  ChannelConfig `json:"discord"`
	Telegram ChannelConfig `json:"telegram"`
}

type ChannelConfig struct {
	Enabled bool `json:"enabled"`
}

type ProviderConfig struct {
	ID           string  `json:"id"`
	Type         string  `json:"type"`
	Enabled      bool    `json:"enabled"`
	DefaultModel string  `json:"default_model"`
	BaseURL      string  `json:"base_url,omitempty"`
	APIKeyEnv    string  `json:"api_key_env,omitempty"`
	Command      string  `json:"command,omitempty"`
	CostPer1KIn  float64 `json:"cost_per_1k_input,omitempty"`
	CostPer1KOut float64 `json:"cost_per_1k_output,omitempty"`
}

type RouterConfig struct {
	DefaultProvider string `json:"default_provider"`
	DefaultModel    string `json:"default_model"`
	CheapFirst      bool   `json:"cheap_first"`
	AllowEscalation bool   `json:"allow_escalation"`
}

type ToolsConfig struct {
	Shell ShellToolConfig `json:"shell"`
	Files FileToolConfig  `json:"files"`
}

type ShellToolConfig struct {
	Enabled bool `json:"enabled"`
}

type FileToolConfig struct {
	Enabled       bool `json:"enabled"`
	WorkspaceOnly bool `json:"workspace_only"`
}

type BudgetConfig struct {
	DailyUSDLimit           float64 `json:"daily_usd_limit"`
	RequireApprovalAboveUSD float64 `json:"require_approval_above_usd"`
}

func DefaultPaths() (Paths, error) {
	dataDir, err := defaultDataDir()
	if err != nil {
		return Paths{}, err
	}

	return Paths{
		DataDir:    dataDir,
		ConfigFile: filepath.Join(dataDir, "config.json"),
		LogDir:     filepath.Join(dataDir, "logs"),
		LogFile:    filepath.Join(dataDir, "logs", "vietclaw.log"),
	}, nil
}

func Default(paths Paths) Config {
	return Config{
		Server: ServerConfig{
			Host: "127.0.0.1",
			Port: 18636,
		},
		Runtime: RuntimeConfig{
			Mode:               "eco",
			MaxConcurrentTasks: 1,
		},
		Database: DatabaseConfig{
			Path: filepath.Join(paths.DataDir, "vietclaw.db"),
		},
		Agent: AgentConfig{
			Name:               "VietClaw",
			Language:           "vi",
			Style:              "natural_short",
			DefaultMode:        "eco",
			Workspace:          filepath.Join(paths.DataDir, "workspace"),
			MaxContextChars:    24000,
			MaxHistoryMessages: 12,
		},
		Channels: ChannelsConfig{
			Discord:  ChannelConfig{Enabled: false},
			Telegram: ChannelConfig{Enabled: false},
		},
		Providers: []ProviderConfig{
			{
				ID:           "mock",
				Type:         "mock",
				Enabled:      true,
				DefaultModel: "mock-small",
			},
		},
		Router: RouterConfig{
			DefaultProvider: "mock",
			DefaultModel:    "mock-small",
			CheapFirst:      true,
			AllowEscalation: true,
		},
		Tools: ToolsConfig{
			Shell: ShellToolConfig{Enabled: false},
			Files: FileToolConfig{
				Enabled:       true,
				WorkspaceOnly: true,
			},
		},
		Budget: BudgetConfig{
			DailyUSDLimit:           0.25,
			RequireApprovalAboveUSD: 0.05,
		},
	}
}

func EnsureDefault(paths Paths) (Config, bool, error) {
	if err := os.MkdirAll(paths.DataDir, 0o755); err != nil {
		return Config{}, false, fmt.Errorf("create data dir: %w", err)
	}
	if err := os.MkdirAll(paths.LogDir, 0o755); err != nil {
		return Config{}, false, fmt.Errorf("create log dir: %w", err)
	}

	cfg, err := Load(paths.ConfigFile)
	if err == nil {
		merged := MergeDefault(cfg, Default(paths))
		if !Equal(cfg, merged) {
			if err := Save(paths.ConfigFile, merged); err != nil {
				return Config{}, false, err
			}
		}
		merged.Database.Path = ExpandPath(merged.Database.Path)
		merged.Agent.Workspace = ExpandPath(merged.Agent.Workspace)
		return merged, false, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return Config{}, false, err
	}

	cfg = Default(paths)
	if err := Save(paths.ConfigFile, cfg); err != nil {
		return Config{}, false, err
	}
	return cfg, true, nil
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}
	cfg.Database.Path = ExpandPath(cfg.Database.Path)
	cfg.Agent.Workspace = ExpandPath(cfg.Agent.Workspace)
	return cfg, nil
}

func Save(path string, cfg Config) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

func ExpandPath(path string) string {
	if path == "~" {
		home, err := os.UserHomeDir()
		if err == nil {
			return home
		}
	}

	if len(path) >= 2 && path[:2] == "~"+string(os.PathSeparator) {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[2:])
		}
	}

	return path
}

func MergeDefault(cfg Config, def Config) Config {
	if cfg.Server.Host == "" {
		cfg.Server.Host = def.Server.Host
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = def.Server.Port
	}
	if cfg.Runtime.Mode == "" {
		cfg.Runtime.Mode = def.Runtime.Mode
	}
	if cfg.Runtime.MaxConcurrentTasks == 0 {
		cfg.Runtime.MaxConcurrentTasks = def.Runtime.MaxConcurrentTasks
	}
	if cfg.Database.Path == "" {
		cfg.Database.Path = def.Database.Path
	}
	if cfg.Agent.Name == "" {
		cfg.Agent.Name = def.Agent.Name
	}
	if cfg.Agent.Language == "" {
		cfg.Agent.Language = def.Agent.Language
	}
	if cfg.Agent.Style == "" {
		cfg.Agent.Style = def.Agent.Style
	}
	if cfg.Agent.DefaultMode == "" {
		cfg.Agent.DefaultMode = def.Agent.DefaultMode
	}
	if cfg.Agent.Workspace == "" {
		cfg.Agent.Workspace = def.Agent.Workspace
	}
	if cfg.Agent.MaxContextChars == 0 {
		cfg.Agent.MaxContextChars = def.Agent.MaxContextChars
	}
	if cfg.Agent.MaxHistoryMessages == 0 {
		cfg.Agent.MaxHistoryMessages = def.Agent.MaxHistoryMessages
	}
	if cfg.Providers == nil || len(cfg.Providers) == 0 {
		cfg.Providers = def.Providers
	}
	if cfg.Router.DefaultProvider == "" {
		cfg.Router.DefaultProvider = def.Router.DefaultProvider
	}
	if cfg.Router.DefaultModel == "" {
		cfg.Router.DefaultModel = def.Router.DefaultModel
	}
	if !cfg.Router.CheapFirst {
		cfg.Router.CheapFirst = def.Router.CheapFirst
	}
	if !cfg.Router.AllowEscalation {
		cfg.Router.AllowEscalation = def.Router.AllowEscalation
	}
	if !cfg.Tools.Files.Enabled {
		cfg.Tools.Files.Enabled = def.Tools.Files.Enabled
	}
	if !cfg.Tools.Files.WorkspaceOnly {
		cfg.Tools.Files.WorkspaceOnly = def.Tools.Files.WorkspaceOnly
	}
	if cfg.Budget.DailyUSDLimit == 0 {
		cfg.Budget.DailyUSDLimit = def.Budget.DailyUSDLimit
	}
	if cfg.Budget.RequireApprovalAboveUSD == 0 {
		cfg.Budget.RequireApprovalAboveUSD = def.Budget.RequireApprovalAboveUSD
	}
	return cfg
}

func Equal(a, b Config) bool {
	left, err := json.Marshal(a)
	if err != nil {
		return false
	}
	right, err := json.Marshal(b)
	if err != nil {
		return false
	}
	return string(left) == string(right)
}

func defaultDataDir() (string, error) {
	if runtime.GOOS == "windows" {
		configDir, err := os.UserConfigDir()
		if err == nil && configDir != "" {
			return filepath.Join(configDir, "VietClaw"), nil
		}
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, ".vietclaw"), nil
}
