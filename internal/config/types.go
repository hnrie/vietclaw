package config

type Paths struct {
	DataDir    string
	ConfigFile string
	LogDir     string
	LogFile    string
}

type Config struct {
	Server    ServerConfig         `json:"server"`
	Runtime   RuntimeConfig        `json:"runtime"`
	Database  DatabaseConfig       `json:"database"`
	Agent     AgentConfig          `json:"agent"`
	Channels  ChannelsConfig       `json:"channels"`
	Providers []ProviderConfig     `json:"providers"`
	Router    RouterConfig         `json:"router"`
	Tools     ToolsConfig          `json:"tools"`
	Budget    BudgetConfig         `json:"budget"`
	Agents    []AgentProfileConfig `json:"agents,omitempty"`
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
	Name               string   `json:"name"`
	Language           string   `json:"language"`
	Style              string   `json:"style"`
	DefaultMode        string   `json:"default_mode"`
	Workspace          string   `json:"workspace"`
	SkillDirs          []string `json:"skill_dirs"`
	MaxContextChars    int      `json:"max_context_chars"`
	MaxHistoryMessages int      `json:"max_history_messages"`
	MaxSteps           int      `json:"max_steps"`
	MaxOutputTokens    int      `json:"max_output_tokens"`
}

type AgentProfileConfig struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Language    string   `json:"language"`
	Persona     string   `json:"persona"`
	Tools       []string `json:"tools,omitempty"`
	Providers   []string `json:"providers,omitempty"`
	MemoryScope string   `json:"memory_scope"`
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
	IntentMode      string `json:"intent_mode"`
	AgentRouting    string `json:"agent_routing"`
	CheapFirst      bool   `json:"cheap_first"`
	AllowEscalation bool   `json:"allow_escalation"`
}

type ToolsConfig struct {
	Shell ShellToolConfig   `json:"shell"`
	Files FileToolConfig    `json:"files"`
	MCP   []MCPServerConfig `json:"mcp,omitempty"`
}

type MCPServerConfig struct {
	ID             string            `json:"id"`
	Enabled        bool              `json:"enabled"`
	Transport      string            `json:"transport,omitempty"`
	URL            string            `json:"url,omitempty"`
	Command        string            `json:"command,omitempty"`
	Args           []string          `json:"args,omitempty"`
	Env            map[string]string `json:"env,omitempty"`
	TimeoutSeconds int               `json:"timeout_seconds,omitempty"`
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
