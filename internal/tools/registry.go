package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"vietclaw/internal/config"
	"vietclaw/internal/providers"
)

type ToolRegistry struct {
	policy Policy
	cfg    config.Config
	tools  map[string]Tool
}

func NewRegistry(cfg config.Config) *ToolRegistry {
	p := NewPolicy(cfg)
	r := &ToolRegistry{
		policy: p,
		cfg:    cfg,
		tools:  make(map[string]Tool),
	}
	r.tools["file_read"] = FileRead{Policy: p}
	r.tools["file_write"] = FileWrite{Policy: p}
	r.tools["shell_exec"] = ShellExec{Policy: p}
	return r
}

func (r *ToolRegistry) Execute(ctx context.Context, name string, argsJSON string) (string, error) {
	normalized := name
	if name == "file.read" {
		normalized = "file_read"
	} else if name == "file.write" {
		normalized = "file_write"
	} else if name == "shell.exec" {
		normalized = "shell_exec"
	}

	t, ok := r.tools[normalized]
	if !ok {
		return "", fmt.Errorf("tool not found: %s", name)
	}

	switch normalized {
	case "file_read":
		var args struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return t.Run(ctx, argsJSON)
		}
		return t.Run(ctx, args.Path)

	case "file_write":
		var args struct {
			Path    string `json:"path"`
			Content string `json:"content"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return "", err
		}
		return t.Run(ctx, args.Path+"\n"+args.Content)

	case "shell_exec":
		var args struct {
			Command string `json:"command"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return t.Run(ctx, argsJSON)
		}
		return t.Run(ctx, args.Command)

	default:
		return t.Run(ctx, argsJSON)
	}
}

func (r *ToolRegistry) GetDefinitions() []providers.ToolDefinition {
	var list []providers.ToolDefinition

	if r.cfg.Tools.Files.Enabled {
		list = append(list, providers.ToolDefinition{
			Type: "function",
			Function: providers.FunctionDetail{
				Name:        "file_read",
				Description: "Đọc nội dung của một tệp tin trong workspace. Trả về toàn bộ nội dung.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"path": map[string]any{
							"type":        "string",
							"description": "Đường dẫn tuyệt đối hoặc tương đối của tệp tin.",
						},
					},
					"required": []string{"path"},
				},
			},
		})

		list = append(list, providers.ToolDefinition{
			Type: "function",
			Function: providers.FunctionDetail{
				Name:        "file_write",
				Description: "Ghi nội dung mới vào một tệp tin trong workspace. Tự động tạo các thư mục cha nếu chưa có.",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"path": map[string]any{
							"type":        "string",
							"description": "Đường dẫn tuyệt đối hoặc tương đối của tệp tin.",
						},
						"content": map[string]any{
							"type":        "string",
							"description": "Nội dung cần ghi vào file.",
						},
					},
					"required": []string{"path", "content"},
				},
			},
		})
	}

	if r.cfg.Tools.Shell.Enabled {
		list = append(list, providers.ToolDefinition{
			Type: "function",
			Function: providers.FunctionDetail{
				Name:        "shell_exec",
				Description: "Thực thi lệnh shell hệ thống và trả về kết quả output kết hợp (stdout + stderr).",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"command": map[string]any{
							"type":        "string",
							"description": "Lệnh command cần thực thi.",
						},
					},
					"required": []string{"command"},
				},
			},
		})
	}

	return list
}
