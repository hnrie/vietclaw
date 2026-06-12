# Tools reference

Agent gọi tools qua LLM function calling. Registry nằm ở `internal/tools/registry.go`; policy kiểm soát quyền truy cập file/shell/network.

## Policy tổng quan

| Policy | Điều kiện |
| --- | --- |
| File tools | `tools.files.enabled` + path trong workspace nếu `workspace_only: true` |
| Shell | `tools.shell.enabled` (tắt mặc định) |
| HTTP tools | Network policy khi shell/network tools chạy |
| Profile allowlist | Agent profile `tools[]` giới hạn subset |

Path alias: `workspace/...` hoặc `.` resolve về `{agent.workspace}`.

---

## Core file tools

| Tool | Mô tả | Params chính |
| --- | --- | --- |
| `file_read` | Đọc nội dung file | `path` |
| `file_write` | Ghi/ghi đè file | `path`, `content` |
| `dir_list` | Liệt kê thư mục | `path` |
| `file_grep` | Tìm pattern trong file | `path`, `pattern` |
| `file_find` | Tìm file theo glob | `pattern`, `path` |
| `file_stat` | Metadata file | `path` |
| `file_head` | N dòng đầu | `path`, `lines` |
| `file_tail` | N dòng cuối | `path`, `lines` |
| `path_info` | Thông tin path (abs, ext, …) | `path` |

---

## Web tools

| Tool | Mô tả |
| --- | --- |
| `web_search` | Tìm kiếm web (open-websearch integration) |
| `web_fetch` | Fetch URL, trả text/markdown |
| `http_request` | HTTP request tùy chỉnh (method, headers, body) — chịu network policy |

---

## Shell & system

| Tool | Mô tả | Ghi chú |
| --- | --- | --- |
| `shell_exec` | Chạy shell command | **Tắt mặc định**; sandbox `none` hoặc `docker` |
| `system_info` | OS, arch, hostname | Read-only |
| `process_list` | Liệt kê processes | |
| `network_ping` | Ping host | |
| `env_get` | Đọc env var | |
| `dns_lookup` | DNS resolve | |
| `ip_lookup` | IP geolocation/info | |

### Shell sandbox modes

| `sandbox` | Hành vi |
| --- | --- |
| `none` | Chạy trực tiếp trên host (nguy hiểm nếu bật trên máy không tin cậy) |
| `docker` | Chạy trong container `docker_image`, network `docker_network` |

`workspace_mode`: `ro` (read-only mount) hoặc `rw`.

### Network policy (shell)

Mặc định chặn:

- `localhost`, metadata endpoints (`169.254.169.254`, …)
- Private IP ranges khi `deny_private: true`

---

## Memory tools

Chỉ available khi `agent.memory_tools.enabled: true`.

| Tool | Mô tả |
| --- | --- |
| `memory_recall` | Tìm memories liên quan query |
| `memory_store` | Lưu fact mới (kind, confidence) |

Agent cũng có **rule-based shortcuts** cho intent `memory_add` / `memory_query` (không qua full LLM loop) khi message match pattern tiếng Việt/Anh — xem [Agents](agents.md).

---

## Utility tools

| Tool | Mô tả |
| --- | --- |
| `hash_calc` | Hash file/content |
| `json_format` | Format JSON |
| `json_validate` | Validate JSON |
| `json_query` | JSONPath-style query |
| `math_calc` | Biểu thức số học |
| `time_current` | Thời gian hiện tại |
| `timestamp_parse` | Parse timestamp |
| `timestamp_format` | Format timestamp |
| `string_transform` | Transform chuỗi |
| `uuid_generate` | UUID v4 |
| `random_string` | Random string |
| `regex_extract` | Regex extract |
| `regex_replace` | Regex replace |
| `text_stats` | Thống kê text |
| `markdown_to_text` | Strip markdown |
| `html_to_text` | Strip HTML |
| `csv_preview` | Preview CSV |
| `csv_to_json` | CSV → JSON |
| `url_parse` | Parse URL components |

---

## Framework tools

| Tool | Mô tả | Điều kiện |
| --- | --- | --- |
| `agent_delegate` | Spawn child agent run | `framework.delegate_enabled` + profile tồn tại |

**Params:** `agent_id`, `message`

Child run dùng session `{parent_session}:delegate:{agent_id}` và `parent_run_id` tracing trong `agent_runs`.

---

## MCP tools

Cấu hình trong `tools.mcp[]`. Lúc khởi tạo registry, VietClaw discover tools từ MCP servers và đăng ký với prefix/id.

MCP tools execute qua `MCPClient` — tên tool normalized khi gọi.

Ví dụ config (stdio):

```json
{
  "tools": {
    "mcp": [
      {
        "id": "filesystem",
        "enabled": true,
        "command": "npx",
        "args": ["-y", "@modelcontextprotocol/server-filesystem", "/path/to/allowed"],
        "timeout_seconds": 30
      }
    ]
  }
}
```

---

## Tool events & reflexion

Mỗi tool call ghi vào bảng `tool_events` (session, input, output, ok/error).

Khi tool **fail** và `agent.reflexion.enabled`:

1. Tạo lesson text: `Tool {name} failed: {err}. Input: {args}`
2. Lưu vào memory kind `experience` với embedding
3. Lesson được inject vào context các lần chat sau (prefix `[lesson]`)

---

## Bảo mật — checklist

1. Giữ `tools.shell.enabled: false` trên máy shared/VPS công cộng
2. Giữ `tools.files.workspace_only: true` trừ khi cố ý cho agent đọc toàn disk
3. Không add bot vào group Discord/Telegram không tin cậy nếu shell bật
4. Review `tools.mcp` — MCP server có quyền tương đương process spawn
5. Dùng `agents[].tools` allowlist cho profile ít đặc quyền (researcher vs coder)
