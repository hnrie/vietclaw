package main

import (
	"fmt"
	"os"

	"vietclaw/internal/version"
)

const (
	cmdVersion  = "version"
	cmdInit     = "init"
	cmdDaemon   = "daemon"
	cmdStatus   = "status"
	cmdDoctor   = "doctor"
	cmdChat     = "chat"
	cmdMemory   = "memory"
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
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	version.Set(buildVersion, buildCommit)

	if len(args) < 2 {
		printUsage()
		return nil
	}

	switch args[1] {
	case cmdVersion:
		return runVersion()
	case cmdInit:
		return runInit()
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
	fmt.Println("usage: vietclaw <version|init|daemon|status|doctor|chat|memory|channels|discord|telegram>")
}
