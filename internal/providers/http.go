package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"vietclaw/internal/config"
)

type CustomHTTP struct {
	providerBase
	client *http.Client
}

func NewCustomHTTP(cfg config.ProviderConfig, client *http.Client) *CustomHTTP {
	return &CustomHTTP{providerBase: providerBase{cfg: cfg}, client: client}
}

func (p *CustomHTTP) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	baseURL := strings.TrimRight(p.cfg.BaseURL, "/")
	if baseURL == "" {
		return ChatResponse{Provider: p.ID(), Model: req.Model, RawError: "missing base_url"}, fmt.Errorf("missing base_url")
	}
	body, err := json.Marshal(req)
	if err != nil {
		return ChatResponse{}, err
	}
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
	inTokens := EstimateMessagesTokens(req.Messages)
	outTokens := req.MaxOutputTokens
	return CostEstimate{
		InputTokens:      inTokens,
		OutputTokens:     outTokens,
		EstimatedCostUSD: EstimateCostUSD(inTokens, outTokens, p.cfg),
	}
}

func (p *CustomHTTP) ChatStream(ctx context.Context, req ChatRequest) (<-chan StreamChunk, error) {
	ch := make(chan StreamChunk, 2)
	go func() {
		defer close(ch)
		resp, err := p.Chat(ctx, req)
		if err != nil {
			ch <- StreamChunk{Error: err.Error()}
			return
		}
		ch <- StreamChunk{Text: resp.Text, ToolCalls: resp.ToolCalls}
		ch <- StreamChunk{Done: true}
	}()
	return ch, nil
}

func (p *CustomHTTP) Embed(ctx context.Context, text string) ([]float32, error) {
	return nil, fmt.Errorf("embeddings not supported by CustomHTTP provider")
}
