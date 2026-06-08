package tools_test

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"vietclaw/internal/config"
	"vietclaw/internal/tools"
)

func TestMCPToolDiscoveryAndExecution(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string `json:"method"`
			Params struct {
				Name      string         `json:"name"`
				Arguments map[string]any `json:"arguments"`
			} `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatal(err)
		}
		switch req.Method {
		case "tools/list":
			_, _ = fmt.Fprint(w, `{"jsonrpc":"2.0","id":1,"result":{"tools":[{"name":"echo","description":"Echo text","inputSchema":{"type":"object","properties":{"text":{"type":"string"}}}}]}}`)
		case "tools/call":
			_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":1,"result":{"content":[{"type":"text","text":"echo:%s"}]}}`, req.Params.Arguments["text"])
		default:
			t.Fatalf("unexpected method: %s", req.Method)
		}
	}))
	defer server.Close()

	cfg := config.Default(config.Paths{DataDir: t.TempDir()})
	cfg.Tools.MCP = []config.MCPServerConfig{{ID: "test", Enabled: true, URL: server.URL}}

	registry := tools.NewRegistry(cfg)
	defs := registry.GetDefinitions()
	found := false
	for _, def := range defs {
		if def.Function.Name == "mcp_test_echo" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected mcp tool definition, got %#v", defs)
	}

	out, err := registry.Execute(context.Background(), "mcp_test_echo", `{"text":"hello"}`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "echo:hello") {
		t.Fatalf("unexpected mcp output: %q", out)
	}
}

func TestMCPStdioToolDiscoveryAndExecution(t *testing.T) {
	cfg := config.Default(config.Paths{DataDir: t.TempDir()})
	cfg.Tools.MCP = []config.MCPServerConfig{{
		ID:        "stdio",
		Enabled:   true,
		Transport: "stdio",
		Command:   os.Args[0],
		Args:      []string{"-test.run=TestMCPStdioHelperProcess", "--"},
		Env:       map[string]string{"GO_WANT_MCP_HELPER_PROCESS": "1"},
	}}

	registry := tools.NewRegistry(cfg)
	defs := registry.GetDefinitions()
	found := false
	for _, def := range defs {
		if def.Function.Name == "mcp_stdio_echo" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected stdio mcp tool definition, got %#v", defs)
	}

	out, err := registry.Execute(context.Background(), "mcp_stdio_echo", `{"text":"hello"}`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "stdio:hello") {
		t.Fatalf("unexpected stdio mcp output: %q", out)
	}
}

func TestMCPStdioHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_MCP_HELPER_PROCESS") != "1" {
		return
	}
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		var req struct {
			ID     int    `json:"id"`
			Method string `json:"method"`
			Params struct {
				Name      string         `json:"name"`
				Arguments map[string]any `json:"arguments"`
			} `json:"params"`
		}
		if err := json.Unmarshal(scanner.Bytes(), &req); err != nil {
			fmt.Fprintf(os.Stdout, `{"jsonrpc":"2.0","id":0,"error":{"message":"bad json"}}`+"\n")
			continue
		}
		switch req.Method {
		case "initialize":
			fmt.Fprintf(os.Stdout, `{"jsonrpc":"2.0","id":%d,"result":{"protocolVersion":"2025-06-18","capabilities":{},"serverInfo":{"name":"helper","version":"test"}}}`+"\n", req.ID)
		case "notifications/initialized":
		case "tools/list":
			fmt.Fprintf(os.Stdout, `{"jsonrpc":"2.0","id":%d,"result":{"tools":[{"name":"echo","description":"Echo text","inputSchema":{"type":"object","properties":{"text":{"type":"string"}}}}]}}`+"\n", req.ID)
		case "tools/call":
			fmt.Fprintf(os.Stdout, `{"jsonrpc":"2.0","id":%d,"result":{"content":[{"type":"text","text":"stdio:%s"}]}}`+"\n", req.ID, req.Params.Arguments["text"])
		default:
			fmt.Fprintf(os.Stdout, `{"jsonrpc":"2.0","id":%d,"error":{"message":"unexpected method"}}`+"\n", req.ID)
		}
	}
	os.Exit(0)
}
