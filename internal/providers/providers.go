package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"vietclaw/internal/config"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	SessionID       string         `json:"session_id"`
	Messages        []Message      `json:"messages"`
	Model           string         `json:"model"`
	Temperature     float64        `json:"temperature"`
	MaxOutputTokens int            `json:"max_output_tokens"`
	Metadata        map[string]any `json:"metadata,omitempty"`
}

type ChatResponse struct {
	Text             string  `json:"text"`
	Provider         string  `json:"provider"`
	Model            string  `json:"model"`
	InputTokens      int     `json:"input_tokens"`
	OutputTokens     int     `json:"output_tokens"`
	EstimatedCostUSD float64 `json:"estimated_cost_usd"`
	RawError         string  `json:"raw_error,omitempty"`
}

type CostEstimate struct {
	InputTokens      int
	OutputTokens     int
	EstimatedCostUSD float64
}

type Provider interface {
	ID() string
	Type() string
	Chat(ctx context.Context, req ChatRequest) (ChatResponse, error)
	EstimateCost(req ChatRequest) CostEstimate
}

func New(cfg config.ProviderConfig) Provider {
	switch cfg.Type {
	case "openai-compatible", "openai":
		return &OpenAICompatible{cfg: cfg, client: &http.Client{Timeout: 60 * time.Second}}
	case "http":
		return &CustomHTTP{cfg: cfg, client: &http.Client{Timeout: 60 * time.Second}}
	case "opencode-cli":
		return &OpenCodeCLI{cfg: cfg}
	default:
		return &Mock{cfg: cfg}
	}
}

func Enabled(configs []config.ProviderConfig) []Provider {
	var out []Provider
	for _, cfg := range configs {
		if cfg.Enabled {
			out = append(out, New(cfg))
		}
	}
	if len(out) == 0 {
		out = append(out, New(config.ProviderConfig{ID: "mock", Type: "mock", Enabled: true, DefaultModel: "mock-small"}))
	}
	return out
}

func Redact(configs []config.ProviderConfig) []config.ProviderConfig {
	redacted := make([]config.ProviderConfig, 0, len(configs))
	for _, cfg := range configs {
		if cfg.APIKeyEnv != "" {
			cfg.APIKeyEnv = cfg.APIKeyEnv
		}
		redacted = append(redacted, cfg)
	}
	return redacted
}

type Mock struct {
	cfg config.ProviderConfig
}

func (m *Mock) ID() string {
	return defaultString(m.cfg.ID, "mock")
}

func (m *Mock) Type() string {
	return "mock"
}

func (m *Mock) Chat(_ context.Context, req ChatRequest) (ChatResponse, error) {
	text := "t là VietClaw, agent runtime nhẹ để điều phối model, memory và tools."
	if len(req.Messages) > 0 {
		last := strings.ToLower(req.Messages[len(req.Messages)-1].Content)
		if strings.Contains(last, "memory") || strings.Contains(last, "nhớ") {
			text = "mock đây: t có thể lưu và tìm memory qua SQLite."
		}
	}
	outTokens := estimateTokens(text)
	inTokens := estimateMessagesTokens(req.Messages)
	return ChatResponse{
		Text:             text,
		Provider:         m.ID(),
		Model:            defaultString(req.Model, defaultString(m.cfg.DefaultModel, "mock-small")),
		InputTokens:      inTokens,
		OutputTokens:     outTokens,
		EstimatedCostUSD: 0,
	}, nil
}

func (m *Mock) EstimateCost(req ChatRequest) CostEstimate {
	return CostEstimate{InputTokens: estimateMessagesTokens(req.Messages), OutputTokens: req.MaxOutputTokens, EstimatedCostUSD: 0}
}

type OpenAICompatible struct {
	cfg    config.ProviderConfig
	client *http.Client
}

func (p *OpenAICompatible) ID() string {
	return p.cfg.ID
}

func (p *OpenAICompatible) Type() string {
	return p.cfg.Type
}

func (p *OpenAICompatible) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	apiKey := os.Getenv(p.cfg.APIKeyEnv)
	if p.cfg.APIKeyEnv != "" && apiKey == "" {
		return ChatResponse{Provider: p.ID(), Model: req.Model, RawError: "missing api key env"}, fmt.Errorf("missing api key env %s", p.cfg.APIKeyEnv)
	}

	body := map[string]any{
		"model":       defaultString(req.Model, p.cfg.DefaultModel),
		"messages":    req.Messages,
		"temperature": req.Temperature,
	}
	if req.MaxOutputTokens > 0 {
		body["max_tokens"] = req.MaxOutputTokens
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return ChatResponse{}, err
	}

	baseURL := strings.TrimRight(p.cfg.BaseURL, "/")
	if baseURL == "" {
		return ChatResponse{Provider: p.ID(), Model: req.Model, RawError: "missing base_url"}, fmt.Errorf("missing base_url")
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v1/chat/completions", bytes.NewReader(encoded))
	if err != nil {
		return ChatResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return ChatResponse{Provider: p.ID(), Model: req.Model, RawError: err.Error()}, err
	}
	defer resp.Body.Close()

	var payload struct {
		Choices []struct {
			Message Message `json:"message"`
		} `json:"choices"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
		} `json:"usage"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return ChatResponse{Provider: p.ID(), Model: req.Model, RawError: "decode response failed"}, err
	}
	if resp.StatusCode >= 400 {
		msg := sanitizeError(payload.Error.Message)
		return ChatResponse{Provider: p.ID(), Model: req.Model, RawError: msg}, fmt.Errorf("provider returned %s: %s", resp.Status, msg)
	}
	if len(payload.Choices) == 0 {
		return ChatResponse{Provider: p.ID(), Model: req.Model, RawError: "empty choices"}, fmt.Errorf("empty provider response")
	}

	inputTokens := payload.Usage.PromptTokens
	outputTokens := payload.Usage.CompletionTokens
	if inputTokens == 0 {
		inputTokens = estimateMessagesTokens(req.Messages)
	}
	if outputTokens == 0 {
		outputTokens = estimateTokens(payload.Choices[0].Message.Content)
	}
	return ChatResponse{
		Text:             payload.Choices[0].Message.Content,
		Provider:         p.ID(),
		Model:            defaultString(req.Model, p.cfg.DefaultModel),
		InputTokens:      inputTokens,
		OutputTokens:     outputTokens,
		EstimatedCostUSD: estimateCost(inputTokens, outputTokens, p.cfg),
	}, nil
}

func (p *OpenAICompatible) EstimateCost(req ChatRequest) CostEstimate {
	inTokens := estimateMessagesTokens(req.Messages)
	outTokens := req.MaxOutputTokens
	if outTokens == 0 {
		outTokens = 512
	}
	return CostEstimate{InputTokens: inTokens, OutputTokens: outTokens, EstimatedCostUSD: estimateCost(inTokens, outTokens, p.cfg)}
}

type CustomHTTP struct {
	cfg    config.ProviderConfig
	client *http.Client
}

func (p *CustomHTTP) ID() string   { return p.cfg.ID }
func (p *CustomHTTP) Type() string { return p.cfg.Type }

func (p *CustomHTTP) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	baseURL := strings.TrimRight(p.cfg.BaseURL, "/")
	if baseURL == "" {
		return ChatResponse{Provider: p.ID(), Model: req.Model, RawError: "missing base_url"}, fmt.Errorf("missing base_url")
	}
	body, _ := json.Marshal(req)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL, bytes.NewReader(body))
	if err != nil {
		return ChatResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := p.client.Do(httpReq)
	if err != nil {
		return ChatResponse{Provider: p.ID(), Model: req.Model, RawError: err.Error()}, err
	}
	defer resp.Body.Close()
	var out ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return ChatResponse{Provider: p.ID(), Model: req.Model, RawError: "decode response failed"}, err
	}
	out.Provider = defaultString(out.Provider, p.ID())
	out.Model = defaultString(out.Model, defaultString(req.Model, p.cfg.DefaultModel))
	return out, nil
}

func (p *CustomHTTP) EstimateCost(req ChatRequest) CostEstimate {
	inTokens := estimateMessagesTokens(req.Messages)
	outTokens := req.MaxOutputTokens
	return CostEstimate{InputTokens: inTokens, OutputTokens: outTokens, EstimatedCostUSD: estimateCost(inTokens, outTokens, p.cfg)}
}

type OpenCodeCLI struct {
	cfg config.ProviderConfig
}

func (p *OpenCodeCLI) ID() string   { return p.cfg.ID }
func (p *OpenCodeCLI) Type() string { return p.cfg.Type }

func (p *OpenCodeCLI) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	if p.cfg.Command == "" {
		return ChatResponse{Provider: p.ID(), Model: req.Model, RawError: "missing command"}, fmt.Errorf("missing opencode command")
	}
	if _, err := exec.LookPath(p.cfg.Command); err != nil {
		return ChatResponse{Provider: p.ID(), Model: req.Model, RawError: "command not found"}, fmt.Errorf("opencode command not found: %s", p.cfg.Command)
	}
	input := ""
	if len(req.Messages) > 0 {
		input = req.Messages[len(req.Messages)-1].Content
	}
	cmd := exec.CommandContext(ctx, p.cfg.Command, input)
	out, err := cmd.Output()
	if err != nil {
		return ChatResponse{Provider: p.ID(), Model: req.Model, RawError: sanitizeError(err.Error())}, err
	}
	text := strings.TrimSpace(string(out))
	return ChatResponse{
		Text:         text,
		Provider:     p.ID(),
		Model:        defaultString(req.Model, p.cfg.DefaultModel),
		InputTokens:  estimateMessagesTokens(req.Messages),
		OutputTokens: estimateTokens(text),
	}, nil
}

func (p *OpenCodeCLI) EstimateCost(req ChatRequest) CostEstimate {
	return CostEstimate{InputTokens: estimateMessagesTokens(req.Messages), OutputTokens: req.MaxOutputTokens}
}

func estimateMessagesTokens(messages []Message) int {
	total := 0
	for _, msg := range messages {
		total += estimateTokens(msg.Content)
	}
	return total
}

func estimateTokens(text string) int {
	n := len([]rune(text)) / 4
	if n < 1 && text != "" {
		return 1
	}
	return n
}

func estimateCost(inTokens, outTokens int, cfg config.ProviderConfig) float64 {
	return (float64(inTokens)/1000)*cfg.CostPer1KIn + (float64(outTokens)/1000)*cfg.CostPer1KOut
}

func sanitizeError(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 300 {
		return value[:300]
	}
	return value
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
