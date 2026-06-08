package agent

import (
	"context"

	"vietclaw/internal/providers"
	"vietclaw/internal/router"
)

const (
	defaultTemperature = 0.2
)

func (s *Service) handleProviderChat(ctx context.Context, req ChatRequest, runID string, intent router.Intent) (ChatResponse, error) {
	embedder := s.router.SelectDefaultEmbedder()
	messages, err := s.context.Messages(ctx, req.SessionID, req.UserID, req.Message, embedder)
	if err != nil {
		_ = s.finishRun(ctx, runID, RunStatusFailed, err.Error(), "", "")
		return ChatResponse{}, err
	}

	chatReq := providers.ChatRequest{
		SessionID:       req.SessionID,
		Messages:        messages,
		Temperature:     defaultTemperature,
		MaxOutputTokens: s.maxOutputTokens(),
		Metadata: map[string]any{
			"user_id":  req.UserID,
			"channel":  req.Channel,
			"mode":     req.Mode,
			"language": s.Language(),
		},
	}
	selection, err := s.router.Select(ctx, chatReq, nil)
	if err != nil {
		reply := err.Error()
		_ = s.addMessage(ctx, req.SessionID, RoleAssistant, reply)
		_ = s.finishRun(ctx, runID, RunStatusNeedsApproval, reply, "", "")
		return ChatResponse{
			OK:        false,
			SessionID: req.SessionID,
			Intent:    string(intent),
			Reply:     reply,
			Error:     reply,
		}, nil
	}
	chatReq.Model = selection.Model

	providerResp, err := selection.Provider.Chat(ctx, chatReq)
	if err != nil {
		_ = s.finishRun(ctx, runID, RunStatusFailed, providerResp.RawError, providerResp.Provider, providerResp.Model)
		return ChatResponse{
			OK:        false,
			SessionID: req.SessionID,
			Intent:    string(intent),
			Provider:  providerResp.Provider,
			Model:     providerResp.Model,
			Error:     providerResp.RawError,
		}, err
	}

	_ = s.addMessage(ctx, req.SessionID, RoleAssistant, providerResp.Text)
	_ = s.insertCost(ctx, providerResp)
	_ = s.finishRun(ctx, runID, RunStatusCompleted, providerResp.Text, providerResp.Provider, providerResp.Model)
	return ChatResponse{
		OK:        true,
		SessionID: req.SessionID,
		Intent:    string(intent),
		Reply:     providerResp.Text,
		Provider:  providerResp.Provider,
		Model:     providerResp.Model,
		CostUSD:   providerResp.EstimatedCostUSD,
	}, nil
}
