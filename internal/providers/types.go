package providers

import (
	"context"

	"vietclaw/internal/config"
)

const (
	TypeMock             = "mock"
	TypeOpenAICompatible = "openai-compatible"
	TypeOpenAI           = "openai"
	TypeAnthropic        = "anthropic"
	TypeCustomHTTP       = "http"
	TypeOpenCodeCLI      = "opencode-cli"

	DefaultMockID    = "mock"
	DefaultMockModel = "mock-small"
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

type providerBase struct {
	cfg config.ProviderConfig
}

func (p providerBase) ID() string {
	return defaultString(p.cfg.ID, DefaultMockID)
}

func (p providerBase) Type() string {
	return p.cfg.Type
}
