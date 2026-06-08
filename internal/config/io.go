package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

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
