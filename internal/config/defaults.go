package config

import "path/filepath"

const (
	AppName = "VietClaw"

	ConfigFileName = "config.json"
	DatabaseName   = "vietclaw.db"
	LogDirName     = "logs"
	LogFileName    = "vietclaw.log"
	WorkspaceName  = "workspace"

	DefaultHost               = "127.0.0.1"
	DefaultPort               = 18636
	DefaultRuntimeMode        = "eco"
	DefaultMaxConcurrentTasks = 1

	DefaultAgentLanguage          = "vi"
	DefaultAgentStyle             = "natural_short"
	DefaultMaxContextChars        = 24000
	DefaultMaxHistoryMessages     = 12
	DefaultMaxAgentSteps          = 5
	DefaultMaxOutputTokens        = 512
	DefaultDiscordTokenEnv        = "VIETCLAW_DISCORD_TOKEN"
	DefaultTelegramTokenEnv       = "VIETCLAW_TELEGRAM_TOKEN"
	DefaultRespondMentionOrReply  = "mention_or_reply"
	DefaultTelegramPollTimeoutSec = 30

	DefaultProviderID    = "mock"
	DefaultProviderType  = "mock"
	DefaultProviderModel = "mock-small"
	DefaultEmbedModel    = "text-embedding-3-small"

	DefaultDailyUSDLimit           = 0.25
	DefaultRequireApprovalAboveUSD = 0.05
)

func Default(paths Paths) Config {
	return Config{
		Server: ServerConfig{
			Host: DefaultHost,
			Port: DefaultPort,
		},
		Runtime: RuntimeConfig{
			Mode:               DefaultRuntimeMode,
			MaxConcurrentTasks: DefaultMaxConcurrentTasks,
		},
		Database: DatabaseConfig{
			Path: filepath.Join(paths.DataDir, DatabaseName),
		},
		Agent: AgentConfig{
			Name:               AppName,
			Language:           DefaultAgentLanguage,
			Style:              DefaultAgentStyle,
			DefaultMode:        DefaultRuntimeMode,
			Workspace:          filepath.Join(paths.DataDir, WorkspaceName),
			MaxContextChars:    DefaultMaxContextChars,
			MaxHistoryMessages: DefaultMaxHistoryMessages,
			MaxSteps:           DefaultMaxAgentSteps,
			MaxOutputTokens:    DefaultMaxOutputTokens,
		},
		Channels: ChannelsConfig{
			Discord: DiscordConfig{
				Enabled:         false,
				TokenEnv:        DefaultDiscordTokenEnv,
				AllowedGuilds:   []string{},
				AllowedChannels: []string{},
				RespondInGuilds: DefaultRespondMentionOrReply,
				RespondInDM:     true,
			},
			Telegram: TelegramConfig{
				Enabled:            false,
				TokenEnv:           DefaultTelegramTokenEnv,
				AllowedChats:       []string{},
				RespondInGroups:    DefaultRespondMentionOrReply,
				RespondInPrivate:   true,
				PollTimeoutSeconds: DefaultTelegramPollTimeoutSec,
			},
		},
		Providers: []ProviderConfig{
			{
				ID:           DefaultProviderID,
				Type:         DefaultProviderType,
				Enabled:      true,
				DefaultModel: DefaultProviderModel,
			},
		},
		Router: RouterConfig{
			DefaultProvider: DefaultProviderID,
			DefaultModel:    DefaultProviderModel,
			CheapFirst:      true,
			AllowEscalation: true,
		},
		Tools: ToolsConfig{
			Shell: ShellToolConfig{Enabled: false},
			Files: FileToolConfig{
				Enabled:       true,
				WorkspaceOnly: true,
			},
		},
		Budget: BudgetConfig{
			DailyUSDLimit:           DefaultDailyUSDLimit,
			RequireApprovalAboveUSD: DefaultRequireApprovalAboveUSD,
		},
	}
}
