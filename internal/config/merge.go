package config

func MergeDefault(cfg Config, def Config) Config {
	if cfg.Server.Host == "" {
		cfg.Server.Host = def.Server.Host
	}
	if cfg.Server.Port == 0 {
		cfg.Server.Port = def.Server.Port
	}
	if cfg.Runtime.Mode == "" {
		cfg.Runtime.Mode = def.Runtime.Mode
	}
	if cfg.Runtime.MaxConcurrentTasks == 0 {
		cfg.Runtime.MaxConcurrentTasks = def.Runtime.MaxConcurrentTasks
	}
	if cfg.Database.Path == "" {
		cfg.Database.Path = def.Database.Path
	}
	cfg.Agent = mergeAgent(cfg.Agent, def.Agent)
	cfg.Channels = mergeChannels(cfg.Channels, def.Channels)
	if cfg.Providers == nil || len(cfg.Providers) == 0 {
		cfg.Providers = def.Providers
	}
	cfg.Router = mergeRouter(cfg.Router, def.Router)
	cfg.Tools = mergeTools(cfg.Tools, def.Tools)
	if cfg.Budget.DailyUSDLimit == 0 {
		cfg.Budget.DailyUSDLimit = def.Budget.DailyUSDLimit
	}
	if cfg.Budget.RequireApprovalAboveUSD == 0 {
		cfg.Budget.RequireApprovalAboveUSD = def.Budget.RequireApprovalAboveUSD
	}
	if cfg.Agents == nil {
		cfg.Agents = def.Agents
	}
	return cfg
}

func mergeAgent(cfg AgentConfig, def AgentConfig) AgentConfig {
	if cfg.Name == "" {
		cfg.Name = def.Name
	}
	if cfg.Language == "" {
		cfg.Language = def.Language
	}
	if cfg.Style == "" {
		cfg.Style = def.Style
	}
	if cfg.DefaultMode == "" {
		cfg.DefaultMode = def.DefaultMode
	}
	if cfg.Workspace == "" {
		cfg.Workspace = def.Workspace
	}
	if cfg.SkillDirs == nil {
		cfg.SkillDirs = def.SkillDirs
	}
	if cfg.MaxContextChars == 0 {
		cfg.MaxContextChars = def.MaxContextChars
	}
	if cfg.MaxHistoryMessages == 0 {
		cfg.MaxHistoryMessages = def.MaxHistoryMessages
	}
	if cfg.MaxSteps == 0 {
		cfg.MaxSteps = def.MaxSteps
	}
	if cfg.MaxOutputTokens == 0 {
		cfg.MaxOutputTokens = def.MaxOutputTokens
	}
	return cfg
}

func mergeChannels(cfg ChannelsConfig, def ChannelsConfig) ChannelsConfig {
	cfg.Discord = mergeDiscord(cfg.Discord, def.Discord)
	cfg.Telegram = mergeTelegram(cfg.Telegram, def.Telegram)
	return cfg
}

func mergeDiscord(cfg DiscordConfig, def DiscordConfig) DiscordConfig {
	if cfg.TokenEnv == "" {
		cfg.TokenEnv = def.TokenEnv
	}
	if cfg.AllowedGuilds == nil {
		cfg.AllowedGuilds = def.AllowedGuilds
	}
	if cfg.AllowedChannels == nil {
		cfg.AllowedChannels = def.AllowedChannels
	}
	if cfg.RespondInGuilds == "" {
		cfg.RespondInGuilds = def.RespondInGuilds
	}
	if !cfg.RespondInDM {
		cfg.RespondInDM = def.RespondInDM
	}
	return cfg
}

func mergeTelegram(cfg TelegramConfig, def TelegramConfig) TelegramConfig {
	if cfg.TokenEnv == "" {
		cfg.TokenEnv = def.TokenEnv
	}
	if cfg.AllowedChats == nil {
		cfg.AllowedChats = def.AllowedChats
	}
	if cfg.RespondInGroups == "" {
		cfg.RespondInGroups = def.RespondInGroups
	}
	if !cfg.RespondInPrivate {
		cfg.RespondInPrivate = def.RespondInPrivate
	}
	if cfg.PollTimeoutSeconds == 0 {
		cfg.PollTimeoutSeconds = def.PollTimeoutSeconds
	}
	return cfg
}

func mergeRouter(cfg RouterConfig, def RouterConfig) RouterConfig {
	if cfg.DefaultProvider == "" {
		cfg.DefaultProvider = def.DefaultProvider
	}
	if cfg.DefaultModel == "" {
		cfg.DefaultModel = def.DefaultModel
	}
	if cfg.IntentMode == "" {
		cfg.IntentMode = def.IntentMode
	}
	if cfg.AgentRouting == "" {
		cfg.AgentRouting = def.AgentRouting
	}
	if !cfg.CheapFirst {
		cfg.CheapFirst = def.CheapFirst
	}
	if !cfg.AllowEscalation {
		cfg.AllowEscalation = def.AllowEscalation
	}
	return cfg
}

func mergeTools(cfg ToolsConfig, def ToolsConfig) ToolsConfig {
	cfg.Shell = mergeShell(cfg.Shell, def.Shell)
	if !cfg.Files.Enabled {
		cfg.Files.Enabled = def.Files.Enabled
	}
	if !cfg.Files.WorkspaceOnly {
		cfg.Files.WorkspaceOnly = def.Files.WorkspaceOnly
	}
	if cfg.MCP == nil {
		cfg.MCP = def.MCP
	}
	return cfg
}

func mergeShell(cfg ShellToolConfig, def ShellToolConfig) ShellToolConfig {
	if cfg.Sandbox == "" {
		cfg.Sandbox = def.Sandbox
	}
	if cfg.DockerBinary == "" {
		cfg.DockerBinary = def.DockerBinary
	}
	if cfg.DockerImage == "" {
		cfg.DockerImage = def.DockerImage
	}
	if cfg.DockerNetwork == "" {
		cfg.DockerNetwork = def.DockerNetwork
	}
	if cfg.WorkspaceMode == "" {
		cfg.WorkspaceMode = def.WorkspaceMode
	}
	if cfg.TimeoutSeconds == 0 {
		cfg.TimeoutSeconds = def.TimeoutSeconds
	}
	return cfg
}
