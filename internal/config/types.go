package config

type Paths struct {
	DataDir    string
	ConfigFile string
	LogDir     string
	LogFile    string
}

type Config struct {
	Server    ServerConfig     `json:"server"`
	Runtime   RuntimeConfig    `json:"runtime"`
	Database  DatabaseConfig   `json:"database"`
	Agent     AgentConfig      `json:"agent"`
	Channels  ChannelsConfig   `json:"channels"`
	Providers []ProviderConfig `json:"providers"`
	Router    RouterConfig     `json:"router"`
	Tools     ToolsConfig      `json:"tools"`
	Budget    BudgetConfig     `json:"budget"`
}

type ServerConfig struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

type RuntimeConfig struct {
	Mode               string `json:"mode"`
	MaxConcurrentTasks int    `json:"max_concurrent_tasks"`
}

type DatabaseConfig struct {
	Path string `json:"path"`
}

type AgentConfig struct {
	Name               string `json:"name"`
	Language           string `json:"language"`
	Style              string `json:"style"`
	DefaultMode        string `json:"default_mode"`
	Workspace          string `json:"workspace"`
	MaxContextChars    int    `json:"max_context_chars"`
	MaxHistoryMessages int    `json:"max_history_messages"`
	MaxSteps           int    `json:"max_steps"`
	MaxOutputTokens    int    `json:"max_output_tokens"`
}

type ChannelsConfig struct {
	Discord  DiscordConfig  `json:"discord"`
	Telegram TelegramConfig `json:"telegram"`
}

type DiscordConfig struct {
	Enabled         bool     `json:"enabled"`
	TokenEnv        string   `json:"token_env"`
	AllowedGuilds   []string `json:"allowed_guilds"`
	AllowedChannels []string `json:"allowed_channels"`
	RespondInGuilds string   `json:"respond_in_guilds"`
	RespondInDM     bool     `json:"respond_in_dm"`
}

type TelegramConfig struct {
	Enabled            bool     `json:"enabled"`
	TokenEnv           string   `json:"token_env"`
	AllowedChats       []string `json:"allowed_chats"`
	RespondInGroups    string   `json:"respond_in_groups"`
	RespondInPrivate   bool     `json:"respond_in_private"`
	PollTimeoutSeconds int      `json:"poll_timeout_seconds"`
}

type ProviderConfig struct {
	ID           string  `json:"id"`
	Type         string  `json:"type"`
	Enabled      bool    `json:"enabled"`
	DefaultModel string  `json:"default_model"`
	BaseURL      string  `json:"base_url,omitempty"`
	APIKeyEnv    string  `json:"api_key_env,omitempty"`
	Command      string  `json:"command,omitempty"`
	EmbedModel   string  `json:"embed_model,omitempty"`
	CostPer1KIn  float64 `json:"cost_per_1k_input,omitempty"`
	CostPer1KOut float64 `json:"cost_per_1k_output,omitempty"`
}

type RouterConfig struct {
	DefaultProvider string `json:"default_provider"`
	DefaultModel    string `json:"default_model"`
	CheapFirst      bool   `json:"cheap_first"`
	AllowEscalation bool   `json:"allow_escalation"`
}

type ToolsConfig struct {
	Shell ShellToolConfig `json:"shell"`
	Files FileToolConfig  `json:"files"`
}

type ShellToolConfig struct {
	Enabled bool `json:"enabled"`
}

type FileToolConfig struct {
	Enabled       bool `json:"enabled"`
	WorkspaceOnly bool `json:"workspace_only"`
}

type BudgetConfig struct {
	DailyUSDLimit           float64 `json:"daily_usd_limit"`
	RequireApprovalAboveUSD float64 `json:"require_approval_above_usd"`
}
