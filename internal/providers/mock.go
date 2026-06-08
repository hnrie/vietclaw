package providers

import (
	"context"
	"strings"

	"vietclaw/internal/config"
	"vietclaw/internal/i18n"
)

type Mock struct {
	providerBase
}

func NewMock(cfg config.ProviderConfig) *Mock {
	cfg.Type = TypeMock
	return &Mock{providerBase: providerBase{cfg: cfg}}
}

func (m *Mock) Chat(_ context.Context, req ChatRequest) (ChatResponse, error) {
	lang := metadataLanguage(req.Metadata)
	text := i18n.T(lang, i18n.ProviderMockDefault)
	if len(req.Messages) >= 2 && strings.Contains(req.Messages[0].Content, "Choose one VietClaw agent") {
		last := strings.ToLower(req.Messages[len(req.Messages)-1].Content)
		if strings.Contains(last, "research") || strings.Contains(last, "nghiên cứu") || strings.Contains(last, "nghien cuu") {
			text = `{"agent_id":"researcher","reason":"research task"}`
		} else {
			text = `{"agent_id":"default","reason":"general task"}`
		}
	}
	if len(req.Messages) > 0 {
		last := strings.ToLower(req.Messages[len(req.Messages)-1].Content)
		if strings.Contains(last, "memory") || strings.Contains(last, "nhớ") {
			text = i18n.T(lang, i18n.ProviderMockMemory)
		}
	}
	return ChatResponse{
		Text:             text,
		Provider:         m.ID(),
		Model:            defaultString(req.Model, defaultString(m.cfg.DefaultModel, DefaultMockModel)),
		InputTokens:      EstimateMessagesTokens(req.Messages),
		OutputTokens:     EstimateTokens(text),
		EstimatedCostUSD: 0,
	}, nil
}

func (m *Mock) EstimateCost(req ChatRequest) CostEstimate {
	return CostEstimate{
		InputTokens:      EstimateMessagesTokens(req.Messages),
		OutputTokens:     req.MaxOutputTokens,
		EstimatedCostUSD: 0,
	}
}

func (m *Mock) ChatStream(ctx context.Context, req ChatRequest) (<-chan StreamChunk, error) {
	ch := make(chan StreamChunk, 2)
	go func() {
		defer close(ch)
		resp, err := m.Chat(ctx, req)
		if err != nil {
			ch <- StreamChunk{Error: err.Error()}
			return
		}
		ch <- StreamChunk{Text: resp.Text, ToolCalls: resp.ToolCalls}
		ch <- StreamChunk{Done: true}
	}()
	return ch, nil
}

func (m *Mock) Embed(ctx context.Context, text string) ([]float32, error) {
	return make([]float32, 1536), nil
}
