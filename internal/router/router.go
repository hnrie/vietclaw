package router

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

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

func (r *ModelRouter) Classify(ctx context.Context, message, language string) Intent {
	ruleIntent := Classify(message)
	mode := strings.ToLower(strings.TrimSpace(r.cfg.Router.IntentMode))
	if mode == "" {
		mode = config.DefaultIntentMode
	}
	if mode == "rule" {
		return ruleIntent
	}
	if mode == "hybrid" && ruleIntent != IntentChat && ruleIntent != IntentUnknown {
		return ruleIntent
	}

	provider := r.defaultProvider(nil)
	if provider == nil || provider.Type() == providers.TypeMock {
		return ruleIntent
	}
	intent, err := classifyWithProvider(ctx, provider, r.defaultModel(provider), message, language)
	if err != nil || intent == IntentUnknown {
		return ruleIntent
	}
	return intent
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

func classifyWithProvider(ctx context.Context, provider providers.Provider, model, message, language string) (Intent, error) {
	resp, err := provider.Chat(ctx, providers.ChatRequest{
		Model:           model,
		MaxOutputTokens: 32,
		Temperature:     0,
		Messages: []providers.Message{
			{
				Role: "system",
				Content: "Classify the user message into exactly one intent: memory_add, memory_query, action, chat, unknown. " +
					"Return compact JSON only: {\"intent\":\"...\"}. Language hint: " + language,
			},
			{Role: "user", Content: message},
		},
	})
	if err != nil {
		return IntentUnknown, err
	}
	return parseIntentResponse(resp.Text), nil
}

func parseIntentResponse(text string) Intent {
	var payload struct {
		Intent string `json:"intent"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(text)), &payload); err == nil {
		return ParseIntent(payload.Intent)
	}
	for _, intent := range []Intent{IntentMemoryAdd, IntentMemoryQuery, IntentAction, IntentChat, IntentUnknown} {
		if strings.Contains(strings.ToLower(text), string(intent)) {
			return intent
		}
	}
	return IntentUnknown
}
