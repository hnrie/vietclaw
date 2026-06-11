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
	cmdVersion = "version"
	cmdInit    = "init"
	cmdSetup   = "setup"
	cmdDaemon  = "daemon"
	cmdStatus  = "status"
	cmdDoctor  = "doctor"
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
	case "help", "-h", "--help":
		printUsage()
		return nil
	default:
		return fmt.Errorf("unknown command %q", args[1])
	}
}

func printUsage() {
	fmt.Println(i18n.T(config.DefaultAgentLanguage, i18n.CLIUsage))
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  version   Print version")
	fmt.Println("  init      Create data dir, config, database")
	fmt.Println("  setup     Interactive first-time configuration")
	fmt.Println("  daemon    Start HTTP server and channels")
	fmt.Println("  status    Query running daemon")
	fmt.Println("  doctor    Health checks")
}

func loadEnvFiles() {
	loadEnvFile(".env")
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
