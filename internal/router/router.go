package router

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"vietclaw/internal/config"
	"vietclaw/internal/providers"
)

type Intent string

const (
	IntentMemoryAdd   Intent = "memory_add"
	IntentMemoryQuery Intent = "memory_query"
	IntentChat        Intent = "chat"
	IntentAction      Intent = "action"
	IntentUnknown     Intent = "unknown"
)

func Classify(message string) Intent {
	text := strings.ToLower(strings.TrimSpace(message))
	switch {
	case text == "":
		return IntentUnknown
	case strings.Contains(text, "nhớ là") || strings.Contains(text, "từ nay") || strings.Contains(text, "lưu lại"):
		return IntentMemoryAdd
	case strings.Contains(text, "mày nhớ gì") || strings.Contains(text, "nhớ gì") || strings.Contains(text, "server chính") || strings.Contains(text, "đã lưu"):
		return IntentMemoryQuery
	case strings.HasPrefix(text, "chạy ") || strings.Contains(text, "đọc file") || strings.Contains(text, "ghi file"):
		return IntentAction
	default:
		return IntentChat
	}
}

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

func (r *ModelRouter) Select(ctx context.Context, req providers.ChatRequest) (Selection, error) {
	provider := r.defaultProvider()
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

func (r *ModelRouter) defaultProvider() providers.Provider {
	for _, p := range r.providers {
		if p.ID() == r.cfg.Router.DefaultProvider {
			return p
		}
	}
	if r.cfg.Router.CheapFirst {
		for _, p := range r.providers {
			if p.Type() == "mock" {
				return p
			}
		}
	}
	return r.providers[0]
}

func (r *ModelRouter) defaultModel(provider providers.Provider) string {
	for _, cfg := range r.cfg.Providers {
		if cfg.ID == provider.ID() && cfg.DefaultModel != "" {
			return cfg.DefaultModel
		}
	}
	if r.cfg.Router.DefaultModel != "" {
		return r.cfg.Router.DefaultModel
	}
	return "mock-small"
}

func (r *ModelRouter) needsApproval(ctx context.Context, estimate float64) bool {
	if estimate <= 0 || r.cfg.Budget.RequireApprovalAboveUSD <= 0 {
		return false
	}
	return estimate > r.cfg.Budget.RequireApprovalAboveUSD
}

func (r *ModelRouter) exceedsDailyBudget(ctx context.Context, estimate float64) bool {
	if estimate <= 0 || r.cfg.Budget.DailyUSDLimit <= 0 {
		return false
	}
	return TodayCost(ctx, r.db)+estimate > r.cfg.Budget.DailyUSDLimit
}

func TodayCost(ctx context.Context, db *sql.DB) float64 {
	if db == nil {
		return 0
	}
	start := time.Now().Local().Format("2006-01-02")
	var total sql.NullFloat64
	_ = db.QueryRowContext(ctx, `SELECT COALESCE(SUM(cost_usd), 0) FROM cost_events WHERE substr(created_at, 1, 10) = ?`, start).Scan(&total)
	if total.Valid {
		return total.Float64
	}
	return 0
}
