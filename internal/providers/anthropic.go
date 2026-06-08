package providers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"vietclaw/internal/config"
)

const (
	anthropicMessagesPath = "/v1/messages"
	anthropicVersion      = "2023-06-01"
)

type Anthropic struct {
	providerBase
	client *http.Client
}

func NewAnthropic(cfg config.ProviderConfig, client *http.Client) *Anthropic {
	cfg.Type = TypeAnthropic
	return &Anthropic{providerBase: providerBase{cfg: cfg}, client: client}
}

func (p *Anthropic) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	apiKey := os.Getenv(p.cfg.APIKeyEnv)
	if p.cfg.APIKeyEnv != "" && apiKey == "" {
		return ChatResponse{Provider: p.ID(), Model: req.Model, RawError: "missing api key env"}, fmt.Errorf("missing api key env %s", p.cfg.APIKeyEnv)
	}
	body := anthropicRequestFromChat(req, defaultString(req.Model, p.cfg.DefaultModel))
	encoded, err := json.Marshal(body)
	if err != nil {
		return ChatResponse{}, err
	}
	baseURL := strings.TrimRight(p.cfg.BaseURL, "/")
	if baseURL == "" {
		return ChatResponse{Provider: p.ID(), Model: req.Model, RawError: "missing base_url"}, fmt.Errorf("missing base_url")
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+anthropicMessagesPath, bytes.NewReader(encoded))
	if err != nil {
		return ChatResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("anthropic-version", anthropicVersion)
	if apiKey != "" {
		httpReq.Header.Set("x-api-key", apiKey)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return ChatResponse{Provider: p.ID(), Model: req.Model, RawError: err.Error()}, err
	}
	defer resp.Body.Close()

	var payload anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return ChatResponse{Provider: p.ID(), Model: req.Model, RawError: "decode response failed"}, err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		msg := SanitizeError(payload.Error.Message)
		return ChatResponse{Provider: p.ID(), Model: req.Model, RawError: msg}, fmt.Errorf("provider returned %s: %s", resp.Status, msg)
	}
	text := payload.Text()
	inputTokens := payload.Usage.InputTokens
	outputTokens := payload.Usage.OutputTokens
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
func (p *Anthropic) ChatStream(ctx context.Context, req ChatRequest) (<-chan StreamChunk, error) {
	apiKey := os.Getenv(p.cfg.APIKeyEnv)
	if p.cfg.APIKeyEnv != "" && apiKey == "" {
		return nil, fmt.Errorf("missing api key env %s", p.cfg.APIKeyEnv)
	}
	body := anthropicRequestFromChat(req, defaultString(req.Model, p.cfg.DefaultModel))
	body.Stream = true
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	baseURL := strings.TrimRight(p.cfg.BaseURL, "/")
	if baseURL == "" {
		return nil, fmt.Errorf("missing base_url")
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+anthropicMessagesPath, bytes.NewReader(encoded))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("anthropic-version", anthropicVersion)
	if apiKey != "" {
		httpReq.Header.Set("x-api-key", apiKey)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		defer resp.Body.Close()
		var payload anthropicResponse
		_ = json.NewDecoder(resp.Body).Decode(&payload)
		return nil, fmt.Errorf("provider returned %s: %s", resp.Status, payload.Error.Message)
	}

	ch := make(chan StreamChunk, 32)
	go func() {
		defer resp.Body.Close()
		defer close(ch)
		readAnthropicStream(resp.Body, ch)
		ch <- StreamChunk{Done: true}
	}()
	return ch, nil
}

func (p *Anthropic) EstimateCost(req ChatRequest) CostEstimate {
	inTokens := EstimateMessagesTokens(req.Messages)
	outTokens := defaultOutputTokens(req.MaxOutputTokens)
	return CostEstimate{
		InputTokens:      inTokens,
		OutputTokens:     outTokens,
		EstimatedCostUSD: EstimateCostUSD(inTokens, outTokens, p.cfg),
	}
}

func (p *Anthropic) Embed(ctx context.Context, text string) ([]float32, error) {
	return nil, fmt.Errorf("embeddings not supported by Anthropic provider")
}

func anthropicRequestFromChat(req ChatRequest, model string) anthropicRequest {
	var system []string
	var messages []anthropicMessage
	for _, msg := range req.Messages {
		if msg.Role == "system" {
			system = append(system, msg.Content)
			continue
		}
		role := msg.Role
		if role != "assistant" {
			role = "user"
		}
		messages = append(messages, anthropicMessage{Role: role, Content: msg.Content})
	}
	return anthropicRequest{
		Model:       model,
		System:      strings.Join(system, "\n\n"),
		Messages:    messages,
		MaxTokens:   defaultOutputTokens(req.MaxOutputTokens),
		Temperature: req.Temperature,
	}
}

type anthropicRequest struct {
	Model       string             `json:"model"`
	System      string             `json:"system,omitempty"`
	Messages    []anthropicMessage `json:"messages"`
	MaxTokens   int                `json:"max_tokens"`
	Temperature float64            `json:"temperature,omitempty"`
	Stream      bool               `json:"stream,omitempty"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (r anthropicResponse) Text() string {
	var parts []string
	for _, block := range r.Content {
		if block.Type == "text" && strings.TrimSpace(block.Text) != "" {
			parts = append(parts, block.Text)
		}
	}
	return strings.Join(parts, "\n")
}

func readAnthropicStream(body io.Reader, ch chan<- StreamChunk) {
	reader := bufio.NewReader(body)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				ch <- StreamChunk{Error: err.Error()}
			}
			return
		}
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			return
		}
		var event struct {
			Type  string `json:"type"`
			Delta struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"delta"`
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}
		switch event.Type {
		case "content_block_delta":
			if event.Delta.Text != "" {
				ch <- StreamChunk{Text: event.Delta.Text}
			}
		case "error":
			ch <- StreamChunk{Error: event.Error.Message}
			return
		case "message_stop":
			return
		}
	}
}
