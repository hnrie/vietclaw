package main

import (
	"fmt"
	"os"

	"vietclaw/internal/config"
	"vietclaw/internal/db"
	"vietclaw/internal/logging"
)

func runInit() error {
	paths, err := config.DefaultPaths()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(paths.LogDir, 0o755); err != nil {
		return fmt.Errorf("create log dir: %w", err)
	}

	cfg, createdConfig, err := config.EnsureDefault(paths)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(cfg.Agent.Workspace, 0o755); err != nil {
		return fmt.Errorf("create workspace: %w", err)
	}

	logFile, createdLog, err := logging.EnsureLogFile(paths.LogFile)
	if err != nil {
		return err
	}
	_ = logFile.Close()

	database, err := db.Open(cfg.Database.Path)
	if err != nil {
		return err
	}
	defer database.Close()
	if err := db.ApplySchema(database); err != nil {
		return err
	}

	printCreated("data dir", paths.DataDir, true)
	printCreated("config", paths.ConfigFile, createdConfig)
	printCreated("database", cfg.Database.Path, true)
	printCreated("log file", paths.LogFile, createdLog)
	printCreated("workspace", cfg.Agent.Workspace, true)
	return nil
}

func printCreated(label, path string, created bool) {
	if created {
		fmt.Printf("[ok] %s ready: %s\n", label, path)
		return
	}
	fmt.Printf("[ok] %s exists: %s\n", label, path)
}
