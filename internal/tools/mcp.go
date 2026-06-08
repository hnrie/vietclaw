package tools

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"vietclaw/internal/config"
	"vietclaw/internal/providers"
)

const (
	mcpToolPrefix        = "mcp"
	defaultMCPHTTPClient = 10 * time.Second
	defaultMCPStdioCall  = 15 * time.Second
	mcpTransportHTTP     = "http"
	mcpTransportStdio    = "stdio"
)

type MCPClient struct {
	server config.MCPServerConfig
	client *http.Client
}

func NewMCPClient(server config.MCPServerConfig) *MCPClient {
	return &MCPClient{
		server: server,
		client: &http.Client{Timeout: defaultMCPHTTPClient},
	}
}

type MCPDiscoveredTool struct {
	Name       string
	Definition providers.ToolDefinition
}

func (c *MCPClient) Discover(ctx context.Context) ([]MCPDiscoveredTool, error) {
	var result struct {
		Tools []struct {
			Name        string         `json:"name"`
			Description string         `json:"description"`
			InputSchema map[string]any `json:"inputSchema"`
		} `json:"tools"`
	}
	if err := c.call(ctx, "tools/list", map[string]any{}, &result); err != nil {
		return nil, err
	}
	discovered := make([]MCPDiscoveredTool, 0, len(result.Tools))
	for _, tool := range result.Tools {
		name := mcpToolName(c.server.ID, tool.Name)
		discovered = append(discovered, MCPDiscoveredTool{
			Name: tool.Name,
			Definition: providers.ToolDefinition{
				Type: "function",
				Function: providers.FunctionDetail{
					Name:        name,
					Description: tool.Description,
					Parameters:  tool.InputSchema,
				},
			},
		})
	}
	return discovered, nil
}

func (c *MCPClient) Tools(ctx context.Context) ([]providers.ToolDefinition, error) {
	discovered, err := c.Discover(ctx)
	if err != nil {
		return nil, err
	}
	defs := make([]providers.ToolDefinition, 0, len(discovered))
	for _, tool := range discovered {
		defs = append(defs, tool.Definition)
	}
	return defs, nil
}

func (c *MCPClient) Execute(ctx context.Context, toolName, argsJSON string) (string, error) {
	var args map[string]any
	if strings.TrimSpace(argsJSON) != "" {
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return "", err
		}
	}
	var result struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := c.call(ctx, "tools/call", map[string]any{
		"name":      toolName,
		"arguments": args,
	}, &result); err != nil {
		return "", err
	}
	var parts []string
	for _, item := range result.Content {
		if item.Text != "" {
			parts = append(parts, item.Text)
		}
	}
	return strings.Join(parts, "\n"), nil
}

func (c *MCPClient) call(ctx context.Context, method string, params map[string]any, out any) error {
	transport := strings.ToLower(strings.TrimSpace(c.server.Transport))
	if transport == "" {
		if c.server.Command != "" {
			transport = mcpTransportStdio
		} else {
			transport = mcpTransportHTTP
		}
	}
	if transport == mcpTransportStdio {
		return c.callStdio(ctx, method, params, out)
	}
	return c.callHTTP(ctx, method, params, out)
}

func (c *MCPClient) callHTTP(ctx context.Context, method string, params map[string]any, out any) error {
	if c.server.URL == "" {
		return fmt.Errorf("mcp server %s missing url", c.server.ID)
	}
	body, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  method,
		"params":  params,
	})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.server.URL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		return fmt.Errorf("mcp server %s returned %s", c.server.ID, resp.Status)
	}

	var payload struct {
		Result json.RawMessage `json:"result"`
		Error  struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return err
	}
	if payload.Error.Message != "" {
		return fmt.Errorf("mcp server %s error: %s", c.server.ID, payload.Error.Message)
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(payload.Result, out)
}

func (c *MCPClient) callStdio(ctx context.Context, method string, params map[string]any, out any) error {
	if c.server.Command == "" {
		return fmt.Errorf("mcp server %s missing command", c.server.ID)
	}
	timeout := defaultMCPStdioCall
	if c.server.TimeoutSeconds > 0 {
		timeout = time.Duration(c.server.TimeoutSeconds) * time.Second
	}
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(callCtx, c.server.Command, c.server.Args...)
	if len(c.server.Env) > 0 {
		cmd.Env = append(os.Environ(), envPairs(c.server.Env)...)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	defer func() {
		_ = stdin.Close()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	}()

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 8*1024*1024)
	if err := writeMCPMessage(stdin, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": "2025-06-18",
			"capabilities":    map[string]any{},
			"clientInfo": map[string]any{
				"name":    "vietclaw",
				"version": "dev",
			},
		},
	}); err != nil {
		return err
	}
	if _, err := readMCPResponse(scanner, 1); err != nil {
		return withStderr(c.server.ID, err, stderr.String())
	}
	if err := writeMCPMessage(stdin, map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
		"params":  map[string]any{},
	}); err != nil {
		return err
	}
	if err := writeMCPMessage(stdin, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  method,
		"params":  params,
	}); err != nil {
		return err
	}
	result, err := readMCPResponse(scanner, 2)
	if err != nil {
		return withStderr(c.server.ID, err, stderr.String())
	}
	if out == nil {
		return nil
	}
	return json.Unmarshal(result, out)
}

func writeMCPMessage(stdin io.Writer, payload map[string]any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	data = append(data, '\n')
	_, err = stdin.Write(data)
	return err
}

func readMCPResponse(scanner *bufio.Scanner, id int) (json.RawMessage, error) {
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var payload struct {
			ID     int             `json:"id"`
			Result json.RawMessage `json:"result"`
			Error  struct {
				Message string `json:"message"`
			} `json:"error"`
			Method string `json:"method"`
		}
		if err := json.Unmarshal([]byte(line), &payload); err != nil {
			return nil, err
		}
		if payload.ID != id {
			continue
		}
		if payload.Error.Message != "" {
			return nil, errors.New(payload.Error.Message)
		}
		return payload.Result, nil
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("mcp stdio response %d not found", id)
}

func envPairs(values map[string]string) []string {
	pairs := make([]string, 0, len(values))
	for key, value := range values {
		if strings.TrimSpace(key) == "" {
			continue
		}
		pairs = append(pairs, key+"="+value)
	}
	return pairs
}

func withStderr(serverID string, err error, stderr string) error {
	stderr = strings.TrimSpace(stderr)
	if stderr == "" {
		return err
	}
	return fmt.Errorf("mcp server %s: %w; stderr: %s", serverID, err, stderr)
}

func mcpToolName(serverID, toolName string) string {
	return mcpToolPrefix + "_" + sanitizeToolName(serverID) + "_" + sanitizeToolName(toolName)
}

func sanitizeToolName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer(".", "_", "-", "_", " ", "_", "/", "_", ":", "_")
	return replacer.Replace(value)
}
