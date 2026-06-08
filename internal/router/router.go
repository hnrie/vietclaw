package router

import (
	"context"
	"database/sql"
	"fmt"

	"vietclaw/internal/config"
	"vietclaw/internal/providers"
)

type ModelRouter struct {
	cfg       config.Config
	providers []providers.Provider
	db        *sql.DB
}

type Selection struct {
	Provider providers.Provider
	Model    string
	Estimate providers.CostEstimate
}

func NewModelRouter(cfg config.Config, db *sql.DB, available []providers.Provider) *ModelRouter {
	return &ModelRouter{cfg: cfg, providers: available, db: db}
}

func (r *ModelRouter) Select(ctx context.Context, req providers.ChatRequest, excludeIDs []string) (Selection, error) {
	provider := r.defaultProvider(excludeIDs)
	if provider == nil {
		return Selection{}, fmt.Errorf("no fallback provider available")
	}
	model := r.defaultModel(provider)
	req.Model = model
	estimate := provider.EstimateCost(req)
	if r.needsApproval(ctx, estimate.EstimatedCostUSD) {
		return Selection{}, fmt.Errorf("approval required for estimated cost %.4f USD", estimate.EstimatedCostUSD)
	}
	if r.exceedsDailyBudget(ctx, estimate.EstimatedCostUSD) {
		return Selection{}, fmt.Errorf("daily budget exceeded")
	}
	return Selection{Provider: provider, Model: model, Estimate: estimate}, nil
}

func (r *ModelRouter) SelectDefaultEmbedder() providers.Provider {
	for _, p := range r.providers {
		if p.Type() == providers.TypeOpenAI || p.Type() == providers.TypeOpenAICompatible {
			return p
		}
	}
	return nil
}
