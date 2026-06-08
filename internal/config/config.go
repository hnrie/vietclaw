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
	Server    ServerConfig   `json:"server"`
	Runtime   RuntimeConfig  `json:"runtime"`
	Database  DatabaseConfig `json:"database"`
	Channels  ChannelsConfig `json:"channels"`
	Providers []Provider     `json:"providers"`
	Budget    BudgetConfig   `json:"budget"`
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

type ChannelsConfig struct {
	Discord  ChannelConfig `json:"discord"`
	Telegram ChannelConfig `json:"telegram"`
}

type ChannelConfig struct {
	Enabled bool `json:"enabled"`
}

type Provider struct {
	Name string `json:"name,omitempty"`
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
		Channels: ChannelsConfig{
			Discord:  ChannelConfig{Enabled: false},
			Telegram: ChannelConfig{Enabled: false},
		},
		Providers: []Provider{},
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
		cfg.Database.Path = ExpandPath(cfg.Database.Path)
		return cfg, false, nil
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
