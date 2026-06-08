package agent

import (
	"context"
	"time"

	"vietclaw/internal/providers"
)

func (s *Service) insertCost(ctx context.Context, resp providers.ChatResponse) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `
INSERT INTO cost_events (provider, model, input_tokens, output_tokens, cost_usd, created_at)
VALUES (?, ?, ?, ?, ?, ?)`,
		resp.Provider, resp.Model, resp.InputTokens, resp.OutputTokens, resp.EstimatedCostUSD, now)
	return err
}

func (s *Service) insertRun(ctx context.Context, id, sessionID, intent, provider, model, status, summary string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `
INSERT INTO agent_runs (id, session_id, intent, provider, model, status, summary, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, sessionID, intent, nullable(provider), nullable(model), status, summary, now, now)
	return err
}

func (s *Service) finishRun(ctx context.Context, id, status, summary, provider, model string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `
UPDATE agent_runs SET status = ?, summary = ?, provider = ?, model = ?, updated_at = ? WHERE id = ?`,
		status, summary, nullable(provider), nullable(model), now, id)
	return err
}
