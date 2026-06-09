package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"vietclaw/internal/config"
	"vietclaw/internal/i18n"
	"vietclaw/internal/version"
)

const (
	cmdVersion  = "version"
	cmdInit     = "init"
	cmdSetup    = "setup"
	cmdDaemon   = "daemon"
	cmdStatus   = "status"
	cmdDoctor   = "doctor"
	cmdChat     = "chat"
	cmdMemory   = "memory"
	cmdTools    = "tools"
	cmdChannels = "channels"
	cmdDiscord  = "discord"
	cmdTelegram = "telegram"
)

var (
	buildVersion = "0.1.0"
	buildCommit  = "dev"
)

func main() {
	if err := run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, i18n.T(config.DefaultAgentLanguage, i18n.CLIErrorPrefix), err)
		os.Exit(1)
	}
}

func run(args []string) error {
	version.Set(buildVersion, buildCommit)
	loadEnvFiles()

	if len(args) < 2 {
		printUsage()
		return nil
	}

	switch args[1] {
	case cmdVersion:
		return runVersion()
	case cmdInit:
		return runInit()
	case cmdSetup:
		return runSetup()
	case cmdDaemon:
		return runDaemon()
	case cmdStatus:
		return runStatus()
	case cmdDoctor:
		return runDoctor()
	case cmdChat:
		return runChat(args[2:])
	case cmdMemory:
		return runMemory(args[2:])
	case cmdTools:
		return runTools(args[2:])
	case cmdChannels:
		return runChannels()
	case cmdDiscord:
		return runDiscord(args[2:])
	case cmdTelegram:
		return runTelegram(args[2:])
	case "help", "-h", "--help":
		printUsage()
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[1])
	}
}

func printUsage() {
	fmt.Println(i18n.T(config.DefaultAgentLanguage, i18n.CLIUsage))
}

func loadEnvFiles() {
	// 1. Load from current working directory
	loadEnvFile(".env")

	// 2. Load from data dir
	if paths, err := config.DefaultPaths(); err == nil {
		loadEnvFile(filepath.Join(paths.DataDir, ".env"))
	}
}

func loadEnvFile(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			val = strings.Trim(val, `"'`)
			if os.Getenv(key) == "" {
				_ = os.Setenv(key, val)
			}
		}
	}
}
