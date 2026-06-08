package agent

import (
	"context"
	"strings"

	"vietclaw/internal/i18n"
	"vietclaw/internal/memory"
	"vietclaw/internal/router"
)

func (s *Service) handleMemoryAdd(ctx context.Context, req ChatRequest, runID string, intent router.Intent) (ChatResponse, error) {
	content := cleanMemoryContent(req.Message)
	embedder := s.router.SelectDefaultEmbedder()
	var embedding []float32
	var err error
	if embedder != nil {
		embedding, err = embedder.Embed(ctx, content)
		if err != nil {
			// fallback without embedding but log error
			embedding = nil
		}
	}

	rec, err := s.mem.Add(ctx, memory.Record{
		Scope:      "user:" + req.UserID,
		Kind:       memory.KindNote,
		Content:    content,
		Confidence: memory.ConfidenceConfirmed,
		Embedding:  embedding,
	})
	if err != nil {
		_ = s.finishRun(ctx, runID, RunStatusFailed, err.Error(), ProviderLocal, ModelRule)
		return ChatResponse{}, err
	}

	reply := s.text(i18n.MemorySaved, rec.Content)
	_ = s.addMessage(ctx, req.SessionID, RoleAssistant, reply)
	_ = s.finishRun(ctx, runID, RunStatusCompleted, reply, ProviderLocal, ModelRule)
	return ChatResponse{
		OK:        true,
		SessionID: req.SessionID,
		Intent:    string(intent),
		Reply:     reply,
		Provider:  ProviderLocal,
		Model:     ModelRule,
	}, nil
}

func (s *Service) handleMemoryQuery(ctx context.Context, req ChatRequest, runID string, intent router.Intent) (ChatResponse, error) {
	query := cleanMemoryQuery(req.Message)
	embedder := s.router.SelectDefaultEmbedder()
	records, err := s.mem.SearchHybrid(ctx, "user:"+req.UserID, query, 5, embedder)
	if err != nil {
		_ = s.finishRun(ctx, runID, RunStatusFailed, err.Error(), ProviderLocal, ModelRule)
		return ChatResponse{}, err
	}

	reply := s.text(i18n.MemoryNotFound)
	if len(records) > 0 {
		parts := make([]string, 0, len(records))
		for _, rec := range records {
			parts = append(parts, rec.Content)
		}
		reply = s.text(i18n.MemoryFound, strings.Join(parts, "; "))
	}
	_ = s.addMessage(ctx, req.SessionID, RoleAssistant, reply)
	_ = s.finishRun(ctx, runID, RunStatusCompleted, reply, ProviderLocal, ModelRule)
	return ChatResponse{
		OK:        true,
		SessionID: req.SessionID,
		Intent:    string(intent),
		Reply:     reply,
		Provider:  ProviderLocal,
		Model:     ModelRule,
	}, nil
}
