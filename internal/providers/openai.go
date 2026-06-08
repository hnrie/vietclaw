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
	if len(req.Tools) > 0 {
		body["tools"] = req.Tools
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

	var toolCalls []ToolCall
	if len(payload.Choices[0].Message.ToolCalls) > 0 {
		toolCalls = payload.Choices[0].Message.ToolCalls
	}

	return ChatResponse{
		Text:             text,
		Provider:         p.ID(),
		Model:            defaultString(req.Model, p.cfg.DefaultModel),
		InputTokens:      inputTokens,
		OutputTokens:     outputTokens,
		EstimatedCostUSD: EstimateCostUSD(inputTokens, outputTokens, p.cfg),
		ToolCalls:        toolCalls,
	}, nil
}

func (p *OpenAICompatible) ChatStream(ctx context.Context, req ChatRequest) (<-chan StreamChunk, error) {
	apiKey := os.Getenv(p.cfg.APIKeyEnv)
	if p.cfg.APIKeyEnv != "" && apiKey == "" {
		return nil, fmt.Errorf("missing api key env %s", p.cfg.APIKeyEnv)
	}

	body := map[string]any{
		"model":       defaultString(req.Model, p.cfg.DefaultModel),
		"messages":    req.Messages,
		"temperature": req.Temperature,
		"stream":      true,
	}
	if req.MaxOutputTokens > 0 {
		body["max_tokens"] = req.MaxOutputTokens
	}
	if len(req.Tools) > 0 {
		body["tools"] = req.Tools
	}

	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	baseURL := strings.TrimRight(p.cfg.BaseURL, "/")
	if baseURL == "" {
		return nil, fmt.Errorf("missing base_url")
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+openAIChatCompletionsPath, bytes.NewReader(encoded))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= http.StatusBadRequest {
		defer resp.Body.Close()
		var payload openAIChatResponse
		_ = json.NewDecoder(resp.Body).Decode(&payload)
		return nil, fmt.Errorf("provider returned %s: %s", resp.Status, payload.Error.Message)
	}

	ch := make(chan StreamChunk, 32)
	go func() {
		defer resp.Body.Close()
		defer close(ch)

		reader := bufio.NewReader(resp.Body)
		var toolBuilders []struct {
			id   string
			name string
			args strings.Builder
		}

		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					ch <- StreamChunk{Error: err.Error()}
				}
				break
			}
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if !strings.HasPrefix(line, "data: ") {
				continue
			}
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				break
			}

			var chunk struct {
				Choices []struct {
					Delta struct {
						Content   string `json:"content"`
						ToolCalls []struct {
							Index    int    `json:"index"`
							ID       string `json:"id"`
							Type     string `json:"type"`
							Function struct {
								Name      string `json:"name"`
								Arguments string `json:"arguments"`
							} `json:"function"`
						} `json:"tool_calls"`
					} `json:"delta"`
				} `json:"choices"`
			}
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				continue
			}

			if len(chunk.Choices) == 0 {
				continue
			}

			delta := chunk.Choices[0].Delta
			if delta.Content != "" {
				ch <- StreamChunk{Text: delta.Content}
			}

			for _, tc := range delta.ToolCalls {
				for len(toolBuilders) <= tc.Index {
					toolBuilders = append(toolBuilders, struct {
						id   string
						name string
						args strings.Builder
					}{})
				}
				if tc.ID != "" {
					toolBuilders[tc.Index].id = tc.ID
				}
				if tc.Function.Name != "" {
					toolBuilders[tc.Index].name = tc.Function.Name
				}
				if tc.Function.Arguments != "" {
					toolBuilders[tc.Index].args.WriteString(tc.Function.Arguments)
				}
			}
		}

		var finalToolCalls []ToolCall
		for _, b := range toolBuilders {
			if b.name != "" || b.id != "" {
				finalToolCalls = append(finalToolCalls, ToolCall{
					ID:   b.id,
					Type: "function",
					Function: ToolCallFunction{
						Name:      b.name,
						Arguments: b.args.String(),
					},
				})
			}
		}
		if len(finalToolCalls) > 0 {
			ch <- StreamChunk{ToolCalls: finalToolCalls}
		}
		ch <- StreamChunk{Done: true}
	}()

	return ch, nil
}

func (p *OpenAICompatible) Embed(ctx context.Context, text string) ([]float32, error) {
	apiKey := os.Getenv(p.cfg.APIKeyEnv)
	if p.cfg.APIKeyEnv != "" && apiKey == "" {
		return nil, fmt.Errorf("missing api key env %s", p.cfg.APIKeyEnv)
	}

	body := map[string]any{
		"model": defaultString(p.cfg.EmbedModel, config.DefaultEmbedModel),
		"input": text,
	}

	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	baseURL := strings.TrimRight(p.cfg.BaseURL, "/")
	if baseURL == "" {
		return nil, fmt.Errorf("missing base_url")
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v1/embeddings", bytes.NewReader(encoded))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	if apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	}

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		var payload struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		_ = json.NewDecoder(resp.Body).Decode(&payload)
		return nil, fmt.Errorf("provider returned %s: %s", resp.Status, payload.Error.Message)
	}

	var payload struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	if len(payload.Data) == 0 {
		return nil, fmt.Errorf("empty embedding data returned")
	}

	return payload.Data[0].Embedding, nil
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
