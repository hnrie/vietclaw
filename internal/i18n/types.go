package i18n

type Language string

const (
	LanguageVietnamese Language = "vi"
	LanguageEnglish    Language = "en"
)

type MessageID string

const (
	AgentActionBlocked     MessageID = "agent.action_blocked"
	AgentMessageRequired   MessageID = "agent.message_required"
	MemorySaved            MessageID = "memory.saved"
	MemoryFound            MessageID = "memory.found"
	MemoryNotFound         MessageID = "memory.not_found"
	ChannelEmptyPrompt     MessageID = "channel.empty_prompt"
	ChannelEmptyAgentReply MessageID = "channel.empty_agent_reply"
	ProviderMockDefault    MessageID = "provider.mock.default"
	ProviderMockMemory     MessageID = "provider.mock.memory"
	AgentMaxStepsReached   MessageID = "agent.max_steps_reached"
	AgentToolExecuteError  MessageID = "agent.tool_execute_error"
	AgentToolStatus        MessageID = "agent.tool_status"
	AgentToolResult        MessageID = "agent.tool_result"
	ToolFileRead           MessageID = "tool.file_read"
	ToolFileWrite          MessageID = "tool.file_write"
	ToolShellExec          MessageID = "tool.shell_exec"
	ToolPathParam          MessageID = "tool.param.path"
	ToolContentParam       MessageID = "tool.param.content"
	ToolCommandParam       MessageID = "tool.param.command"
	SystemPromptBase       MessageID = "system.prompt_base"
	SystemMemoryHeader     MessageID = "system.memory_header"
	SystemHistoryHeader    MessageID = "system.history_header"
	CLIUsage               MessageID = "cli.usage"
	CLIErrorPrefix         MessageID = "cli.error_prefix"
	CLIMemorySaved         MessageID = "cli.memory_saved"
)
