package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"vietclaw/internal/config"
)

const openAIChatCompletionsPath = "/v1/chat/completions"

type OpenAICompatible struct {
	providerBase
	client *http.Client
}

func NewOpenAICompatible(cfg config.ProviderConfig, client *http.Client) *OpenAICompatible {
	return &OpenAICompatible{providerBase: providerBase{cfg: cfg}, client: client}
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
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+openAIChatCompletionsPath, bytes.NewReader(encoded))
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

	var payload openAIChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return ChatResponse{Provider: p.ID(), Model: req.Model, RawError: "decode response failed"}, err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		msg := SanitizeError(payload.Error.Message)
		return ChatResponse{Provider: p.ID(), Model: req.Model, RawError: msg}, fmt.Errorf("provider returned %s: %s", resp.Status, msg)
	}
	if len(payload.Choices) == 0 {
		return ChatResponse{Provider: p.ID(), Model: req.Model, RawError: "empty choices"}, fmt.Errorf("empty provider response")
	}

	text := payload.Choices[0].Message.Content
	inputTokens := payload.Usage.PromptTokens
	outputTokens := payload.Usage.CompletionTokens
	if inputTokens == 0 {
		inputTokens = EstimateMessagesTokens(req.Messages)
	}
	if outputTokens == 0 {
		outputTokens = EstimateTokens(text)
	}
	return ChatResponse{
		Text:             text,
		Provider:         p.ID(),
		Model:            defaultString(req.Model, p.cfg.DefaultModel),
		InputTokens:      inputTokens,
		OutputTokens:     outputTokens,
		EstimatedCostUSD: EstimateCostUSD(inputTokens, outputTokens, p.cfg),
	}, nil
}

func (p *OpenAICompatible) EstimateCost(req ChatRequest) CostEstimate {
	inTokens := EstimateMessagesTokens(req.Messages)
	outTokens := defaultOutputTokens(req.MaxOutputTokens)
	return CostEstimate{
		InputTokens:      inTokens,
		OutputTokens:     outTokens,
		EstimatedCostUSD: EstimateCostUSD(inTokens, outTokens, p.cfg),
	}
}

type openAIChatResponse struct {
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
