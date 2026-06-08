package main

import (
	"testing"

	"vietclaw/internal/config"
)

func TestUpdateChannelEnabledKeepsExistingConfig(t *testing.T) {
	cfg := config.Default(config.Paths{DataDir: t.TempDir()})
	cfg.Server.Port = 19000
	cfg.Agent.Name = "CustomClaw"

	updated, err := updateChannelEnabled(cfg, "discord", true)
	if err != nil {
		t.Fatal(err)
	}
	if !updated.Channels.Discord.Enabled {
		t.Fatal("discord was not enabled")
	}
	if updated.Channels.Telegram.Enabled {
		t.Fatal("telegram should stay disabled")
	}
	if updated.Server.Port != 19000 || updated.Agent.Name != "CustomClaw" {
		t.Fatalf("unrelated config changed: %#v", updated)
	}

	updated, err = updateChannelEnabled(updated, "discord", false)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Channels.Discord.Enabled {
		t.Fatal("discord was not disabled")
	}
}
