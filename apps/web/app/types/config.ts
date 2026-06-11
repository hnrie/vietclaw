export type ServerConfig = {
  host: string
  port: number
}

export type FrameworkConfig = {
  enabled: boolean
  delegate_enabled: boolean
  hooks_enabled: boolean
}

export type RuntimeConfig = {
  mode: string
  max_concurrent_tasks: number
}

export type DatabaseConfig = {
  path: string
}

export type ReflexionConfig = {
  enabled: boolean
}

export type HeartbeatConfig = {
  enabled: boolean
  interval_seconds: number
  session_id: string
  user_id: string
  prompt: string
}

export type MemoryToolsConfig = {
  enabled: boolean
}

export type AgentConfig = {
  experience: string
  name: string
  language: string
  style: string
  default_mode: string
  workspace: string
  skill_dirs: string[]
  max_context_chars: number
  max_history_messages: number
  max_steps: number
  max_output_tokens: number
  reflexion: ReflexionConfig
  heartbeat: HeartbeatConfig
  memory_tools: MemoryToolsConfig
}

export type AttachmentConfig = {
  enabled: boolean
  max_files: number
  max_bytes: number
  allowed_extensions: string[]
}

export type DiscordConfig = {
  enabled: boolean
  token_env: string
  allowed_guilds: string[]
  allowed_channels: string[]
  respond_in_guilds: string
  respond_in_dm: boolean
}

export type TelegramConfig = {
  enabled: boolean
  token_env: string
  allowed_chats: string[]
  respond_in_groups: string
  respond_in_private: boolean
  poll_timeout_seconds: number
}

export type ChannelsConfig = {
  discord: DiscordConfig
  telegram: TelegramConfig
  attachments: AttachmentConfig
}

export type ProviderConfigFull = {
  id: string
  type: string
  enabled: boolean
  default_model: string
  base_url?: string
  api_key_env?: string
  command?: string
  embed_model?: string
  cost_per_1k_input?: number
  cost_per_1k_output?: number
}

export type RouterConfig = {
  default_provider: string
  default_model: string
  intent_mode: string
  agent_routing: string
  cheap_first: boolean
  allow_escalation: boolean
}

export type ShellNetworkPolicyConfig = {
  enabled: boolean
  restrict_to_allow_hosts: boolean
  allow_hosts?: string[]
  deny_hosts?: string[]
  deny_private: boolean
}

export type ShellToolConfig = {
  enabled: boolean
  sandbox?: string
  docker_binary?: string
  docker_image?: string
  docker_network?: string
  workspace_mode?: string
  timeout_seconds?: number
  network_policy?: ShellNetworkPolicyConfig
}

export type FileToolConfig = {
  enabled: boolean
  workspace_only: boolean
}

export type ToolsConfig = {
  shell: ShellToolConfig
  files: FileToolConfig
}

export type BudgetConfig = {
  daily_usd_limit: number
  require_approval_above_usd: number
}

export type AgentProfileConfig = {
  id: string
  name: string
  language: string
  persona: string
  tools?: string[]
  providers?: string[]
  memory_scope: string
  max_steps?: number
}

export type VietClawConfig = {
  server: ServerConfig
  framework: FrameworkConfig
  runtime: RuntimeConfig
  database: DatabaseConfig
  agent: AgentConfig
  channels: ChannelsConfig
  providers: ProviderConfigFull[]
  router: RouterConfig
  tools: ToolsConfig
  budget: BudgetConfig
  agents?: AgentProfileConfig[]
}

export type SettingsPutResponse = {
  ok: boolean
  config: VietClawConfig
  error?: string
}

export type ChannelEnvTest = {
  name: string
  enabled: boolean
  token_env: string
  env_found: boolean
}
