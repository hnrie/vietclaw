package providers

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"vietclaw/internal/config"
)

type OpenCodeCLI struct {
	providerBase
}

func NewOpenCodeCLI(cfg config.ProviderConfig) *OpenCodeCLI {
	cfg.Type = TypeOpenCodeCLI
	return &OpenCodeCLI{providerBase: providerBase{cfg: cfg}}
}

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
		return ChatResponse{Provider: p.ID(), Model: req.Model, RawError: SanitizeError(err.Error())}, err
	}
	text := strings.TrimSpace(string(out))
	return ChatResponse{
		Text:         text,
		Provider:     p.ID(),
		Model:        defaultString(req.Model, p.cfg.DefaultModel),
		InputTokens:  EstimateMessagesTokens(req.Messages),
		OutputTokens: EstimateTokens(text),
	}, nil
}

func (p *OpenCodeCLI) EstimateCost(req ChatRequest) CostEstimate {
	return CostEstimate{
		InputTokens:  EstimateMessagesTokens(req.Messages),
		OutputTokens: req.MaxOutputTokens,
	}
}
