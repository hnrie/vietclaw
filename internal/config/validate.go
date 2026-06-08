package config

import "fmt"

func Validate(cfg Config) error {
	if cfg.Server.Port <= 0 || cfg.Server.Port > 65535 {
		return fmt.Errorf("server.port must be between 1 and 65535")
	}
	if cfg.Runtime.MaxConcurrentTasks < 1 {
		return fmt.Errorf("runtime.max_concurrent_tasks must be >= 1")
	}
	if cfg.Agent.MaxSteps < 1 {
		return fmt.Errorf("agent.max_steps must be >= 1")
	}
	if cfg.Agent.MaxOutputTokens < 1 {
		return fmt.Errorf("agent.max_output_tokens must be >= 1")
	}
	if cfg.Budget.DailyUSDLimit < 0 || cfg.Budget.RequireApprovalAboveUSD < 0 {
		return fmt.Errorf("budget values must be >= 0")
	}
	if cfg.Tools.Shell.Sandbox != "" && cfg.Tools.Shell.Sandbox != "none" && cfg.Tools.Shell.Sandbox != "docker" {
		return fmt.Errorf("tools.shell.sandbox must be none or docker")
	}
	if cfg.Tools.Shell.WorkspaceMode != "" && cfg.Tools.Shell.WorkspaceMode != "ro" && cfg.Tools.Shell.WorkspaceMode != "rw" {
		return fmt.Errorf("tools.shell.workspace_mode must be ro or rw")
	}
	if cfg.Tools.Shell.TimeoutSeconds < 0 {
		return fmt.Errorf("tools.shell.timeout_seconds must be >= 0")
	}
	for _, provider := range cfg.Providers {
		if provider.ID == "" {
			return fmt.Errorf("provider id is required")
		}
		if provider.Type == "" {
			return fmt.Errorf("provider %s type is required", provider.ID)
		}
	}
	for _, profile := range cfg.Agents {
		if profile.ID == "" {
			return fmt.Errorf("agent profile id is required")
		}
	}
	return nil
}
