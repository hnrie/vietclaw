package main

import (
	"fmt"
	"os"

	"vietclaw/internal/agent"
	"vietclaw/internal/config"
	"vietclaw/internal/db"
)

func localAgent() (*agent.Service, func(), error) {
	_, cfg, err := loadOrCreateConfig()
	if err != nil {
		return nil, nil, err
	}
	if err := os.MkdirAll(cfg.Agent.Workspace, 0o755); err != nil {
		return nil, nil, fmt.Errorf("create workspace: %w", err)
	}
	database, err := db.Open(cfg.Database.Path)
	if err != nil {
		return nil, nil, err
	}
	if err := db.ApplySchema(database); err != nil {
		_ = database.Close()
		return nil, nil, err
	}
	return agent.NewService(cfg, database), func() { _ = database.Close() }, nil
}

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
