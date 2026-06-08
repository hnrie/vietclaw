package router_test

import (
	"context"
	"path/filepath"
	"testing"

	"vietclaw/internal/config"
	"vietclaw/internal/db"
	"vietclaw/internal/providers"
	"vietclaw/internal/router"
)

func TestClassify(t *testing.T) {
	tests := map[string]router.Intent{
		"nhớ là server chính dùng Docker": router.IntentMemoryAdd,
		"mày nhớ gì về server chính":      router.IntentMemoryQuery,
		"chạy ls":                         router.IntentAction,
		"mày là gì":                       router.IntentChat,
		"remember that I use SQLite":      router.IntentMemoryAdd,
		"read file config.json":           router.IntentAction,
		"":                                router.IntentUnknown,
	}
	for input, want := range tests {
		if got := router.Classify(input); got != want {
			t.Fatalf("Classify(%q) = %s, want %s", input, got, want)
		}
	}
}

func TestRouterHybridFallsBackToRuleWithMockProvider(t *testing.T) {
	cfg := config.Default(config.Paths{DataDir: t.TempDir()})
	cfg.Router.IntentMode = config.DefaultIntentMode
	r := router.NewModelRouter(cfg, nil, providers.Enabled(cfg.Providers))
	if got := r.Classify(context.Background(), "hello there", "en"); got != router.IntentChat {
		t.Fatalf("hybrid classify = %s", got)
	}
}

func TestBudgetRequiresApproval(t *testing.T) {
	database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if err := db.ApplySchema(database); err != nil {
		t.Fatal(err)
	}

	cfg := config.Default(config.Paths{DataDir: t.TempDir()})
	cfg.Providers = []config.ProviderConfig{{
		ID:           "paid",
		Type:         "openai-compatible",
		Enabled:      true,
		DefaultModel: "paid-small",
		BaseURL:      "http://example.invalid",
		CostPer1KIn:  1,
		CostPer1KOut: 1,
	}}
	cfg.Router.DefaultProvider = "paid"
	cfg.Router.DefaultModel = "paid-small"
	cfg.Budget.RequireApprovalAboveUSD = 0.01

	r := router.NewModelRouter(cfg, database, providers.Enabled(cfg.Providers))
	_, err = r.Select(context.Background(), providers.ChatRequest{
		Messages:        []providers.Message{{Role: "user", Content: "hello"}},
		MaxOutputTokens: 512,
	}, nil)
	if err == nil {
		t.Fatal("expected approval error")
	}
}
