package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"vietclaw/internal/config"
	"vietclaw/internal/tools"
	"vietclaw/internal/websearch"
)

// runWebSearch dispatches `vietclaw websearch ...` subcommands. The
// open-websearch sub project lives at <repo>/open-websearch and exposes web
// search + content fetch tools over the Model Context Protocol. These
// commands manage installing / building / enabling / probing that MCP server
// inside VietClaw's config.
func runWebSearch(args []string) error {
	if len(args) == 0 {
		return websearchUsage()
	}
	switch args[0] {
	case "install":
		return runWebSearchInstall()
	case "build":
		return runWebSearchBuild()
	case "enable":
		return runWebSearchEnable()
	case "disable":
		return runWebSearchDisable()
	case "remove":
		return runWebSearchRemove()
	case "status":
		return runWebSearchStatus()
	case "test":
		return runWebSearchTest(args[1:])
	case "path":
		return runWebSearchPath()
	case "tools":
		return runWebSearchTools()
	case "help", "-h", "--help":
		return websearchUsage()
	default:
		return fmt.Errorf("unknown websearch command %q", args[0])
	}
}

func websearchUsage() error {
	fmt.Println(`vietclaw websearch <command>

Manage the bundled open-websearch MCP sub project at ./open-websearch.

Commands:
  install   Run "npm install" inside the sub project.
  build     Run "npm run build" inside the sub project.
  enable    Build (if needed) and register the MCP server in config.json.
  disable   Mark the MCP server entry disabled (entry stays in config).
  remove    Delete the MCP server entry from config.json.
  status    Print sub project + MCP server status.
  path      Print the resolved sub project directory.
  tools     List the MCP tools exposed by open-websearch.
  test      Spawn the server and discover its tools live.`)
	return nil
}

func resolveSubProject() (string, error) {
	paths, _, err := loadOrCreateConfig()
	if err != nil {
		return "", err
	}
	locator := websearch.NewLocator(paths.DataDir)
	dir, err := locator.Resolve()
	if err != nil {
		return "", err
	}
	return dir, nil
}

func runWebSearchPath() error {
	dir, err := resolveSubProject()
	if err != nil {
		return err
	}
	fmt.Println(dir)
	return nil
}

func runWebSearchInstall() error {
	dir, err := resolveSubProject()
	if err != nil {
		return err
	}
	fmt.Printf("[websearch] npm install in %s\n", dir)
	if err := websearch.Install(context.Background(), dir, os.Stderr); err != nil {
		return err
	}
	fmt.Println("[ok] open-websearch dependencies installed")
	return nil
}

func runWebSearchBuild() error {
	dir, err := resolveSubProject()
	if err != nil {
		return err
	}
	fmt.Printf("[websearch] npm run build in %s\n", dir)
	if err := websearch.EnsureBuilt(context.Background(), dir, os.Stderr); err != nil {
		return err
	}
	fmt.Println("[ok] open-websearch built")
	return nil
}

func runWebSearchEnable() error {
	paths, cfg, err := loadOrCreateConfig()
	if err != nil {
		return err
	}
	dir, err := websearch.NewLocator(paths.DataDir).Resolve()
	if err != nil {
		return err
	}
	fmt.Printf("[websearch] ensuring build in %s\n", dir)
	if err := websearch.EnsureBuilt(context.Background(), dir, os.Stderr); err != nil {
		return err
	}
	server, err := websearch.MCPServerConfig(dir, nil)
	if err != nil {
		return err
	}
	cfg = websearch.UpsertMCPServer(cfg, server)
	if err := config.Save(paths.ConfigFile, cfg); err != nil {
		return err
	}
	fmt.Printf("[ok] open-websearch MCP server registered as %q (transport=stdio)\n", server.ID)
	printWebSearchSummary(server)
	return nil
}

func runWebSearchDisable() error {
	paths, cfg, err := loadOrCreateConfig()
	if err != nil {
		return err
	}
	cfg, found := websearch.SetEnabled(cfg, false)
	if !found {
		return fmt.Errorf("open-websearch MCP server not present in config; run `vietclaw websearch enable`")
	}
	if err := config.Save(paths.ConfigFile, cfg); err != nil {
		return err
	}
	fmt.Println("[ok] open-websearch MCP server disabled")
	return nil
}

func runWebSearchRemove() error {
	paths, cfg, err := loadOrCreateConfig()
	if err != nil {
		return err
	}
	cfg = websearch.RemoveMCPServer(cfg)
	if err := config.Save(paths.ConfigFile, cfg); err != nil {
		return err
	}
	fmt.Println("[ok] open-websearch MCP server removed from config")
	return nil
}

func runWebSearchStatus() error {
	paths, cfg, err := loadOrCreateConfig()
	if err != nil {
		return err
	}
	dir, locErr := websearch.NewLocator(paths.DataDir).Resolve()
	fmt.Println("Open-WebSearch sub project")
	fmt.Println(strings.Repeat("-", 32))
	if locErr != nil {
		fmt.Printf("location: not found (%v)\n", locErr)
	} else {
		fmt.Printf("location: %s\n", dir)
		fmt.Printf("built:    %t (%s)\n", websearch.IsBuilt(dir), websearch.EntryRelPath)
	}
	fmt.Println()
	fmt.Println("MCP server")
	fmt.Println(strings.Repeat("-", 32))
	if server, ok := websearch.Find(cfg); ok {
		fmt.Printf("id:       %s\n", server.ID)
		fmt.Printf("enabled:  %t\n", server.Enabled)
		fmt.Printf("command:  %s %s\n", server.Command, strings.Join(server.Args, " "))
		fmt.Printf("timeout:  %ds\n", server.TimeoutSeconds)
		fmt.Printf("env:      %d vars\n", len(server.Env))
	} else {
		fmt.Println("not registered (run `vietclaw websearch enable`)")
	}
	return nil
}

func runWebSearchTools() error {
	fmt.Println("open-websearch MCP tools:")
	for _, name := range websearch.ToolNames() {
		fmt.Printf("  - %s\n", name)
	}
	return nil
}

func runWebSearchTest(args []string) error {
	_ = args
	paths, _, err := loadOrCreateConfig()
	if err != nil {
		return err
	}
	dir, err := websearch.NewLocator(paths.DataDir).Resolve()
	if err != nil {
		return err
	}
	if err := websearch.EnsureBuilt(context.Background(), dir, os.Stderr); err != nil {
		return err
	}
	server, err := websearch.MCPServerConfig(dir, nil)
	if err != nil {
		return err
	}
	fmt.Printf("[websearch] spawning %s %s\n", server.Command, strings.Join(server.Args, " "))
	client := tools.NewMCPClient(server)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	discovered, err := client.Discover(ctx)
	if err != nil {
		return fmt.Errorf("mcp discover failed: %w", err)
	}
	fmt.Printf("[ok] discovered %d tools from open-websearch MCP server\n", len(discovered))
	for _, tool := range discovered {
		fmt.Printf("  - %s (mcp id: %s)\n", tool.Name, tool.Definition.Function.Name)
	}
	if len(discovered) == 0 {
		return fmt.Errorf("no tools discovered; server returned empty tools/list")
	}
	return nil
}

func printWebSearchSummary(server config.MCPServerConfig) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("  ", "  ")
	_ = enc.Encode(server)
}
