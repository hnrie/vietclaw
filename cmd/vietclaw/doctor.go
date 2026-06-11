package main

import (
	"errors"
	"fmt"
	"os"

	"vietclaw/internal/config"
	"vietclaw/internal/db"
)

const (
	channelDiscord  = "discord"
	channelTelegram = "telegram"
)

func runDoctor() error {
	paths, cfg, err := loadExistingConfig()
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			paths, pathErr := config.DefaultPaths()
			if pathErr != nil {
				return pathErr
			}
			fmt.Printf("[warn] data dir missing: %s\n", paths.DataDir)
			fmt.Printf("[warn] config missing: %s\n", paths.ConfigFile)
			fmt.Println("[warn] daemon not running")
			return nil
		}
		return err
	}

	printPathCheck("data dir", dirExists(paths.DataDir))
	printPathCheck("config", fileExists(paths.ConfigFile))
	checkDatabase(cfg.Database.Path)
	checkRuntime(cfg)
	checkBudget(cfg)
	checkChannelEnv(channelDiscord, cfg.Channels.Discord.Enabled, cfg.Channels.Discord.TokenEnv)
	checkChannelEnv(channelTelegram, cfg.Channels.Telegram.Enabled, cfg.Channels.Telegram.TokenEnv)
	checkPort(cfg)
	checkDaemon(cfg)
	checkFramework(cfg)
	return nil
}

func checkFramework(cfg config.Config) {
	if cfg.Framework.Enabled {
		fmt.Println("[ok] agent framework enabled")
	} else {
		fmt.Println("[warn] agent framework disabled")
	}
	if cfg.Framework.DelegateEnabled {
		fmt.Println("[ok] sub-agent delegation enabled")
	}
	if cfg.Framework.HooksEnabled {
		fmt.Println("[ok] framework hooks enabled")
	}
}

func printPathCheck(label string, ok bool) {
	if ok {
		fmt.Printf("[ok] %s found\n", label)
		return
	}
	fmt.Printf("[warn] %s missing\n", label)
}

func checkDatabase(path string) {
	database, err := db.Open(path)
	if err != nil {
		fmt.Printf("[fail] database open: %v\n", err)
		return
	}
	defer database.Close()
	if err := database.Ping(); err != nil {
		fmt.Printf("[fail] database ping: %v\n", err)
		return
	}
	fmt.Println("[ok] database open")
}

func checkRuntime(cfg config.Config) {
	if validRuntimeMode(cfg.Runtime.Mode) {
		fmt.Println("[ok] runtime mode valid")
	} else {
		fmt.Printf("[fail] runtime mode invalid: %s\n", cfg.Runtime.Mode)
	}
	if cfg.Runtime.MaxConcurrentTasks >= 1 {
		fmt.Println("[ok] max concurrent tasks valid")
	} else {
		fmt.Println("[fail] max concurrent tasks must be >= 1")
	}
}

func checkBudget(cfg config.Config) {
	if cfg.Budget.DailyUSDLimit >= 0 && cfg.Budget.RequireApprovalAboveUSD >= 0 {
		fmt.Println("[ok] budget valid")
		return
	}
	fmt.Println("[fail] budget values must be >= 0")
}

func checkChannelEnv(name string, enabled bool, tokenEnv string) {
	if !enabled {
		fmt.Printf("[ok] %s disabled\n", name)
		return
	}
	if _, ok := os.LookupEnv(tokenEnv); ok {
		fmt.Printf("[ok] %s token env found\n", name)
		return
	}
	fmt.Printf("[warn] %s enabled but env missing: %s\n", name, tokenEnv)
}

func checkPort(cfg config.Config) {
	if portAvailable(cfg.Server.Host, cfg.Server.Port) {
		fmt.Println("[ok] port available")
	} else {
		fmt.Println("[warn] port already in use")
	}
}

func checkDaemon(cfg config.Config) {
	if _, err := fetchStatus(cfg); err != nil {
		fmt.Println("[warn] daemon not running")
	} else {
		fmt.Println("[ok] daemon running")
	}
}
