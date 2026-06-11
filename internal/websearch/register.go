package websearch

import (
	"vietclaw/internal/config"
)

// UpsertMCPServer inserts or updates the open-websearch MCP server entry in
// the supplied config, returning the new config. Existing user overrides for
// other servers are preserved.
func UpsertMCPServer(cfg config.Config, server config.MCPServerConfig) config.Config {
	servers := cfg.Tools.MCP
	if servers == nil {
		servers = []config.MCPServerConfig{}
	}
	updated := false
	for i, existing := range servers {
		if existing.ID == server.ID {
			servers[i] = server
			updated = true
			break
		}
	}
	if !updated {
		servers = append(servers, server)
	}
	cfg.Tools.MCP = servers
	return cfg
}

// RemoveMCPServer removes the open-websearch server entry from the config.
func RemoveMCPServer(cfg config.Config) config.Config {
	if len(cfg.Tools.MCP) == 0 {
		return cfg
	}
	filtered := cfg.Tools.MCP[:0]
	for _, server := range cfg.Tools.MCP {
		if server.ID == ServerID {
			continue
		}
		filtered = append(filtered, server)
	}
	cfg.Tools.MCP = append([]config.MCPServerConfig{}, filtered...)
	return cfg
}

// SetEnabled toggles only the Enabled flag of the open-websearch MCP entry.
// Returns the resulting config and a bool indicating whether the entry was
// found. Callers should typically call UpsertMCPServer first if they want to
// guarantee the entry exists.
func SetEnabled(cfg config.Config, enabled bool) (config.Config, bool) {
	for i, server := range cfg.Tools.MCP {
		if server.ID == ServerID {
			server.Enabled = enabled
			cfg.Tools.MCP[i] = server
			return cfg, true
		}
	}
	return cfg, false
}

// Find returns the current open-websearch entry, if any.
func Find(cfg config.Config) (config.MCPServerConfig, bool) {
	for _, server := range cfg.Tools.MCP {
		if server.ID == ServerID {
			return server, true
		}
	}
	return config.MCPServerConfig{}, false
}
