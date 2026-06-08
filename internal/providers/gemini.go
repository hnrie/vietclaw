package providers

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"vietclaw/internal/config"
)

const (
	defaultGeminiBaseURL = "https://generativelanguage.googleapis.com"
	geminiAPIPrefix      = "/v1beta/models/"
)

type Gemini struct {
	providerBase
	client *http.Client
}

func NewGemini(cfg config.ProviderConfig, client *http.Client) *Gemini {
	cfg.Type = TypeGemini
	return &Gemini{providerBase: providerBase{cfg: cfg}, client: client}
}

func (p *Gemini) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	endpoint, err := p.endpoint(req.Model, "generateContent", false)
	if err != nil {
		return ChatResponse{Provider: p.ID(), Model: req.Model, RawError: err.Error()}, err
	}
	body := geminiRequestFromChat(req)
	encoded, err := json.Marshal(body)
	if err != nil {
		return ChatResponse{}, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(encoded))
	if err != nil {
		return ChatResponse{}, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return ChatResponse{Provider: p.ID(), Model: req.Model, RawError: err.Error()}, err
	}
	defer resp.Body.Close()

	var payload geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return ChatResponse{Provider: p.ID(), Model: req.Model, RawError: "decode response failed"}, err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		msg := SanitizeError(payload.Error.Message)
		return ChatResponse{Provider: p.ID(), Model: req.Model, RawError: msg}, fmt.Errorf("provider returned %s: %s", resp.Status, msg)
	}
	text := payload.Text()
	inputTokens := payload.UsageMetadata.PromptTokenCount
	outputTokens := payload.UsageMetadata.CandidatesTokenCount
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

func (p *Gemini) ChatStream(ctx context.Context, req ChatRequest) (<-chan StreamChunk, error) {
	endpoint, err := p.endpoint(req.Model, "streamGenerateContent", true)
	if err != nil {
		return nil, err
	}
	body := geminiRequestFromChat(req)
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(encoded))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= http.StatusBadRequest {
		defer resp.Body.Close()
		var payload geminiResponse
		_ = json.NewDecoder(resp.Body).Decode(&payload)
		return nil, fmt.Errorf("provider returned %s: %s", resp.Status, payload.Error.Message)
	}

	ch := make(chan StreamChunk, 32)
	go func() {
		defer resp.Body.Close()
		defer close(ch)
		readGeminiStream(resp.Body, ch)
		ch <- StreamChunk{Done: true}
	}()
	return ch, nil
}

func (p *Gemini) Embed(ctx context.Context, text string) ([]float32, error) {
	model := defaultString(p.cfg.EmbedModel, config.DefaultEmbedModel)
	endpoint, err := p.endpoint(model, "embedContent", false)
	if err != nil {
		return nil, err
	}
	body := map[string]any{
		"model": "models/" + strings.TrimPrefix(model, "models/"),
		"content": geminiContent{
			Parts: []geminiPart{{Text: text}},
		},
	}
	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(encoded))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= http.StatusBadRequest {
		var payload geminiResponse
		_ = json.NewDecoder(resp.Body).Decode(&payload)
		return nil, fmt.Errorf("provider returned %s: %s", resp.Status, payload.Error.Message)
	}
	var payload struct {
		Embedding struct {
			Values []float32 `json:"values"`
		} `json:"embedding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	if len(payload.Embedding.Values) == 0 {
		return nil, fmt.Errorf("empty embedding data returned")
	}
	return payload.Embedding.Values, nil
}

func (p *Gemini) EstimateCost(req ChatRequest) CostEstimate {
	inTokens := EstimateMessagesTokens(req.Messages)
	outTokens := defaultOutputTokens(req.MaxOutputTokens)
	return CostEstimate{
		InputTokens:      inTokens,
		OutputTokens:     outTokens,
		EstimatedCostUSD: EstimateCostUSD(inTokens, outTokens, p.cfg),
	}
}

func (p *Gemini) endpoint(model, method string, stream bool) (string, error) {
	apiKey := os.Getenv(p.cfg.APIKeyEnv)
	if p.cfg.APIKeyEnv != "" && apiKey == "" {
		return "", fmt.Errorf("missing api key env %s", p.cfg.APIKeyEnv)
	}
	baseURL := strings.TrimRight(defaultString(p.cfg.BaseURL, defaultGeminiBaseURL), "/")
	model = strings.TrimPrefix(defaultString(model, p.cfg.DefaultModel), "models/")
	if model == "" {
		return "", fmt.Errorf("missing model")
	}
	endpoint := baseURL + geminiAPIPrefix + url.PathEscape(model) + ":" + method
	query := url.Values{}
	if apiKey != "" {
		query.Set("key", apiKey)
	}
	if stream {
		query.Set("alt", "sse")
	}
	if encoded := query.Encode(); encoded != "" {
		endpoint += "?" + encoded
	}
	return endpoint, nil
}

func geminiRequestFromChat(req ChatRequest) geminiRequest {
	var system []geminiPart
	var contents []geminiContent
	for _, msg := range req.Messages {
		if msg.Role == "system" {
			system = append(system, geminiPart{Text: msg.Content})
			continue
		}
		role := "user"
		if msg.Role == "assistant" {
			role = "model"
		}
		contents = append(contents, geminiContent{
			Role:  role,
			Parts: []geminiPart{{Text: msg.Content}},
		})
	}
	return geminiRequest{
		Contents:          contents,
		SystemInstruction: geminiContent{Parts: system},
		GenerationConfig: geminiGenerationConfig{
			Temperature:     req.Temperature,
			MaxOutputTokens: defaultOutputTokens(req.MaxOutputTokens),
		},
	}
}

func readGeminiStream(body io.Reader, ch chan<- StreamChunk) {
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
		var payload geminiResponse
		if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &payload); err != nil {
			continue
		}
		if payload.Error.Message != "" {
			ch <- StreamChunk{Error: payload.Error.Message}
			return
		}
		if text := payload.Text(); text != "" {
			ch <- StreamChunk{Text: text}
		}
	}
}

type geminiRequest struct {
	Contents          []geminiContent        `json:"contents"`
	SystemInstruction geminiContent          `json:"systemInstruction,omitempty"`
	GenerationConfig  geminiGenerationConfig `json:"generationConfig,omitempty"`
}

type geminiGenerationConfig struct {
	Temperature     float64 `json:"temperature,omitempty"`
	MaxOutputTokens int     `json:"maxOutputTokens,omitempty"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiResponse struct {
	Candidates []struct {
		Content geminiContent `json:"content"`
	} `json:"candidates"`
	UsageMetadata struct {
		PromptTokenCount     int `json:"promptTokenCount"`
		CandidatesTokenCount int `json:"candidatesTokenCount"`
	} `json:"usageMetadata"`
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

func (r geminiResponse) Text() string {
	var parts []string
	for _, candidate := range r.Candidates {
		for _, part := range candidate.Content.Parts {
			if strings.TrimSpace(part.Text) != "" {
				parts = append(parts, part.Text)
			}
		}
	}
	return strings.Join(parts, "\n")
}
