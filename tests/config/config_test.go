package config_test

import (
	"path/filepath"
	"testing"

	"vietclaw/internal/config"
)

func TestDefaultIncludesAgentRuntime(t *testing.T) {
	dir := t.TempDir()
	cfg := config.Default(config.Paths{DataDir: dir})

	if cfg.Agent.Name != "VietClaw" {
		t.Fatalf("agent name = %q", cfg.Agent.Name)
	}
	if cfg.Agent.Language != config.DefaultAgentLanguage {
		t.Fatalf("agent language = %q", cfg.Agent.Language)
	}
	if cfg.Agent.MaxSteps != config.DefaultMaxAgentSteps || cfg.Agent.MaxOutputTokens != config.DefaultMaxOutputTokens {
		t.Fatalf("agent loop defaults invalid: %#v", cfg.Agent)
	}
	if len(cfg.Agent.SkillDirs) != 1 || cfg.Agent.SkillDirs[0] != config.DefaultSkillDir {
		t.Fatalf("skill dirs default invalid: %#v", cfg.Agent.SkillDirs)
	}
	if cfg.Agent.Workspace != filepath.Join(dir, "workspace") {
		t.Fatalf("workspace = %q", cfg.Agent.Workspace)
	}
	if len(cfg.Providers) != 1 || cfg.Providers[0].ID != "mock" || !cfg.Providers[0].Enabled {
		t.Fatalf("default mock provider missing: %#v", cfg.Providers)
	}
	if cfg.Router.DefaultProvider != "mock" || cfg.Router.DefaultModel != "mock-small" {
		t.Fatalf("router default invalid: %#v", cfg.Router)
	}
	if cfg.Router.IntentMode != config.DefaultIntentMode {
		t.Fatalf("router intent mode = %q", cfg.Router.IntentMode)
	}
	if cfg.Router.AgentRouting != config.DefaultAgentRouting {
		t.Fatalf("router agent routing = %q", cfg.Router.AgentRouting)
	}
	if len(cfg.Agents) != 1 || cfg.Agents[0].ID != config.DefaultAgentID {
		t.Fatalf("default agents invalid: %#v", cfg.Agents)
	}
	if cfg.Tools.Shell.Enabled {
		t.Fatal("shell must be disabled by default")
	}
	if !cfg.Tools.Files.Enabled || !cfg.Tools.Files.WorkspaceOnly {
		t.Fatalf("file tools default invalid: %#v", cfg.Tools.Files)
	}
	if cfg.Channels.Discord.TokenEnv != "VIETCLAW_DISCORD_TOKEN" || cfg.Channels.Discord.RespondInGuilds != "mention_or_reply" || !cfg.Channels.Discord.RespondInDM {
		t.Fatalf("discord default invalid: %#v", cfg.Channels.Discord)
	}
	if cfg.Channels.Telegram.TokenEnv != "VIETCLAW_TELEGRAM_TOKEN" || cfg.Channels.Telegram.RespondInGroups != "mention_or_reply" || !cfg.Channels.Telegram.RespondInPrivate || cfg.Channels.Telegram.PollTimeoutSeconds != 30 {
		t.Fatalf("telegram default invalid: %#v", cfg.Channels.Telegram)
	}
}

func TestMergeDefaultKeepsExistingValues(t *testing.T) {
	def := config.Default(config.Paths{DataDir: t.TempDir()})
	cfg := config.Config{}
	cfg.Server.Host = "0.0.0.0"

	merged := config.MergeDefault(cfg, def)
	if merged.Server.Host != "0.0.0.0" {
		t.Fatalf("existing host was overwritten: %s", merged.Server.Host)
	}
	if merged.Agent.MaxContextChars == 0 || len(merged.Providers) == 0 {
		t.Fatalf("defaults were not merged: %#v", merged)
	}
}

func TestUpdateChannelEnabledKeepsExistingConfig(t *testing.T) {
	cfg := config.Default(config.Paths{DataDir: t.TempDir()})
	cfg.Server.Port = 19000
	cfg.Agent.Name = "CustomClaw"

	updated, err := config.UpdateChannelEnabled(cfg, config.ChannelDiscord, true)
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

	updated, err = config.UpdateChannelEnabled(updated, config.ChannelDiscord, false)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Channels.Discord.Enabled {
		t.Fatal("discord was not disabled")
	}
}
