package main

import (
	"fmt"
	"os"

	"vietclaw/internal/config"
)

func loadOrCreateConfig() (config.Paths, config.Config, error) {
	paths, err := config.DefaultPaths()
	if err != nil {
		return config.Paths{}, config.Config{}, err
	}
	if err := os.MkdirAll(paths.LogDir, 0o755); err != nil {
		return config.Paths{}, config.Config{}, fmt.Errorf("create log dir: %w", err)
	}
	cfg, _, err := config.EnsureDefault(paths)
	return paths, cfg, err
}

func loadExistingConfig() (config.Paths, config.Config, error) {
	paths, err := config.DefaultPaths()
	if err != nil {
		return config.Paths{}, config.Config{}, err
	}
	cfg, err := config.Load(paths.ConfigFile)
	if err != nil {
		return paths, config.Config{}, err
	}
	return paths, cfg, nil
}
