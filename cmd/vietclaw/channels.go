package main

import (
	"fmt"
	"os"

	"vietclaw/internal/config"
)

const (
	channelDiscord  = "discord"
	channelTelegram = "telegram"
	subcmdEnable    = "enable"
	subcmdDisable   = "disable"
)

func runChannels() error {
	_, cfg, err := loadOrCreateConfig()
	if err != nil {
		return err
	}
	printChannelStatus(channelDiscord, cfg.Channels.Discord.Enabled, cfg.Channels.Discord.TokenEnv)
	printChannelStatus(channelTelegram, cfg.Channels.Telegram.Enabled, cfg.Channels.Telegram.TokenEnv)
	return nil
}

func runDiscord(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("discord command is required: enable|disable")
	}
	return setChannelEnabled(channelDiscord, args[0] == subcmdEnable, args[0])
}

func runTelegram(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("telegram command is required: enable|disable")
	}
	return setChannelEnabled(channelTelegram, args[0] == subcmdEnable, args[0])
}

func setChannelEnabled(name string, enabled bool, command string) error {
	if command != subcmdEnable && command != subcmdDisable {
		return fmt.Errorf("unknown %s command %q", name, command)
	}
	paths, cfg, err := loadOrCreateConfig()
	if err != nil {
		return err
	}
	switch name {
	case channelDiscord, channelTelegram:
		updated, err := updateChannelEnabled(cfg, name, enabled)
		if err != nil {
			return err
		}
		cfg = updated
	default:
		return fmt.Errorf("unknown channel %q", name)
	}
	if err := config.Save(paths.ConfigFile, cfg); err != nil {
		return err
	}
	fmt.Printf("[ok] %s enabled=%t\n", name, enabled)
	if enabled {
		tokenEnv := channelTokenEnv(cfg, name)
		if _, ok := os.LookupEnv(tokenEnv); !ok {
			fmt.Printf("[warn] %s env not set: %s\n", name, tokenEnv)
		}
	}
	return nil
}

func updateChannelEnabled(cfg config.Config, name string, enabled bool) (config.Config, error) {
	switch name {
	case channelDiscord:
		cfg.Channels.Discord.Enabled = enabled
	case channelTelegram:
		cfg.Channels.Telegram.Enabled = enabled
	default:
		return cfg, fmt.Errorf("unknown channel %q", name)
	}
	return cfg, nil
}

func printChannelStatus(name string, enabled bool, tokenEnv string) {
	envStatus := "missing"
	if _, ok := os.LookupEnv(tokenEnv); ok {
		envStatus = "set"
	}
	fmt.Printf("%s enabled=%t token_env=%s env=%s\n", name, enabled, tokenEnv, envStatus)
}

func channelTokenEnv(cfg config.Config, name string) string {
	if name == channelTelegram {
		return cfg.Channels.Telegram.TokenEnv
	}
	return cfg.Channels.Discord.TokenEnv
}
