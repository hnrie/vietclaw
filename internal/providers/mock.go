package providers

import (
	"context"
	"strings"

	"vietclaw/internal/config"
)

const (
	mockDefaultReply = "t là VietClaw, agent runtime nhẹ để điều phối model, memory và tools."
	mockMemoryReply  = "mock đây: t có thể lưu và tìm memory qua SQLite."
)

type Mock struct {
	providerBase
}

func NewMock(cfg config.ProviderConfig) *Mock {
	cfg.Type = TypeMock
	return &Mock{providerBase: providerBase{cfg: cfg}}
}

func (m *Mock) Chat(_ context.Context, req ChatRequest) (ChatResponse, error) {
	text := mockDefaultReply
	if len(req.Messages) > 0 {
		last := strings.ToLower(req.Messages[len(req.Messages)-1].Content)
		if strings.Contains(last, "memory") || strings.Contains(last, "nhớ") {
			text = mockMemoryReply
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
