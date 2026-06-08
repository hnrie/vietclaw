package i18n

import (
	"fmt"
	"strings"
)

var catalog = map[Language]map[MessageID]string{
	LanguageVietnamese: {
		AgentActionBlocked:     "tool action cần policy rõ hơn. shell.exec đang tắt mặc định nếu chưa bật trong config.",
		AgentMessageRequired:   "message là bắt buộc",
		MemorySaved:            "ok, t lưu: %s",
		MemoryFound:            "t nhớ: %s",
		MemoryNotFound:         "t chưa thấy memory nào khớp.",
		ChannelEmptyPrompt:     "gọi t rồi muốn t làm gì?",
		ChannelEmptyAgentReply: "t chưa có gì để trả lời.",
		ProviderMockDefault:    "t là VietClaw, agent runtime nhẹ để điều phối model, memory và tools.",
		ProviderMockMemory:     "mock đây: t có thể lưu và tìm memory qua SQLite.",
		AgentMaxStepsReached:   "Đã đạt số bước thực thi tối đa nhưng chưa có phản hồi cuối cùng.",
		AgentToolExecuteError:  "Lỗi thực thi công cụ: %s",
		AgentToolStatus:        "\n*[Chạy công cụ: %s...]*\n",
		AgentToolResult:        "\n*[Kết quả: %s]*\n",
		ToolFileRead:           "Đọc nội dung của một tệp tin trong workspace. Trả về toàn bộ nội dung.",
		ToolFileWrite:          "Ghi nội dung mới vào một tệp tin trong workspace. Tự động tạo các thư mục cha nếu chưa có.",
		ToolShellExec:          "Thực thi lệnh shell hệ thống và trả về kết quả output kết hợp (stdout + stderr).",
		ToolPathParam:          "Đường dẫn tuyệt đối hoặc tương đối của tệp tin.",
		ToolContentParam:       "Nội dung cần ghi vào file.",
		ToolCommandParam:       "Lệnh command cần thực thi.",
		SystemPromptBase:       "Bạn là %s, agent điều phối nhẹ. Trả lời tiếng Việt ngắn, tự nhiên.",
		SystemMemoryHeader:     "Memory liên quan:",
		SystemHistoryHeader:    "Lịch sử gần đây:",
		CLIUsage:               "usage: vietclaw <version|init|daemon|status|doctor|chat|memory|channels|discord|telegram>",
		CLIErrorPrefix:         "lỗi:",
		CLIMemorySaved:         "ok, t lưu: %s",
	},
	LanguageEnglish: {
		AgentActionBlocked:     "tool action needs a clearer policy. shell.exec is disabled by default unless enabled in config.",
		AgentMessageRequired:   "message is required",
		MemorySaved:            "ok, saved: %s",
		MemoryFound:            "I remember: %s",
		MemoryNotFound:         "I did not find any matching memory.",
		ChannelEmptyPrompt:     "you called me, but what should I do?",
		ChannelEmptyAgentReply: "I do not have anything to reply with yet.",
		ProviderMockDefault:    "I am VietClaw, a lightweight agent runtime for routing models, memory, and tools.",
		ProviderMockMemory:     "mock here: I can save and search memory through SQLite.",
		AgentMaxStepsReached:   "The maximum execution step count was reached without a final response.",
		AgentToolExecuteError:  "Tool execution error: %s",
		AgentToolStatus:        "\n*[Running tool: %s...]*\n",
		AgentToolResult:        "\n*[Result: %s]*\n",
		ToolFileRead:           "Read a file inside the workspace and return its full content.",
		ToolFileWrite:          "Write new content to a file inside the workspace. Parent directories are created when missing.",
		ToolShellExec:          "Run a system shell command and return combined stdout and stderr output.",
		ToolPathParam:          "Absolute or relative file path.",
		ToolContentParam:       "Content to write into the file.",
		ToolCommandParam:       "Command to execute.",
		SystemPromptBase:       "You are %s, a lightweight orchestration agent. Reply in concise, natural English.",
		SystemMemoryHeader:     "Relevant memory:",
		SystemHistoryHeader:    "Recent history:",
		CLIUsage:               "usage: vietclaw <version|init|daemon|status|doctor|chat|memory|channels|discord|telegram>",
		CLIErrorPrefix:         "error:",
		CLIMemorySaved:         "ok, saved: %s",
	},
}

func Normalize(language string) Language {
	switch Language(strings.ToLower(strings.TrimSpace(language))) {
	case LanguageEnglish:
		return LanguageEnglish
	default:
		return LanguageVietnamese
	}
}

func T(language string, id MessageID, args ...any) string {
	lang := Normalize(language)
	template := catalog[lang][id]
	if template == "" {
		template = catalog[LanguageVietnamese][id]
	}
	if template == "" {
		return string(id)
	}
	if len(args) == 0 {
		return template
	}
	return fmt.Sprintf(template, args...)
}
