package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"vietclaw/internal/config"
	"vietclaw/internal/i18n"
	"vietclaw/internal/providers"
)

const (
	toolFileRead  = "file_read"
	toolFileWrite = "file_write"
	toolShellExec = "shell_exec"
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
	r.tools[toolFileRead] = FileRead{Policy: p}
	r.tools[toolFileWrite] = FileWrite{Policy: p}
	r.tools[toolShellExec] = ShellExec{Policy: p}
	return r
}

func (r *ToolRegistry) Execute(ctx context.Context, name string, argsJSON string) (string, error) {
	normalized := normalizeToolName(name)
	t, ok := r.tools[normalized]
	if !ok {
		return "", fmt.Errorf("tool not found: %s", name)
	}

	switch normalized {
	case toolFileRead:
		var args struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return t.Run(ctx, argsJSON)
		}
		return t.Run(ctx, args.Path)
	case toolFileWrite:
		var args struct {
			Path    string `json:"path"`
			Content string `json:"content"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return "", err
		}
		return t.Run(ctx, args.Path+"\n"+args.Content)
	case toolShellExec:
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
	lang := r.cfg.Agent.Language

	if r.cfg.Tools.Files.Enabled {
		list = append(list, providers.ToolDefinition{
			Type: "function",
			Function: providers.FunctionDetail{
				Name:        toolFileRead,
				Description: i18n.T(lang, i18n.ToolFileRead),
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"path": map[string]any{
							"type":        "string",
							"description": i18n.T(lang, i18n.ToolPathParam),
						},
					},
					"required": []string{"path"},
				},
			},
		})

		list = append(list, providers.ToolDefinition{
			Type: "function",
			Function: providers.FunctionDetail{
				Name:        toolFileWrite,
				Description: i18n.T(lang, i18n.ToolFileWrite),
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"path": map[string]any{
							"type":        "string",
							"description": i18n.T(lang, i18n.ToolPathParam),
						},
						"content": map[string]any{
							"type":        "string",
							"description": i18n.T(lang, i18n.ToolContentParam),
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
				Name:        toolShellExec,
				Description: i18n.T(lang, i18n.ToolShellExec),
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"command": map[string]any{
							"type":        "string",
							"description": i18n.T(lang, i18n.ToolCommandParam),
						},
					},
					"required": []string{"command"},
				},
			},
		})
	}

	return list
}

func normalizeToolName(name string) string {
	switch name {
	case "file.read":
		return toolFileRead
	case "file.write":
		return toolFileWrite
	case "shell.exec":
		return toolShellExec
	default:
		return name
	}
}
