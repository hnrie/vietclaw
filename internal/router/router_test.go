package router

import (
	"context"
	"path/filepath"
	"testing"

	"vietclaw/internal/config"
	"vietclaw/internal/db"
	"vietclaw/internal/providers"
)

func TestClassify(t *testing.T) {
	tests := map[string]Intent{
		"nhớ là server chính dùng Docker": IntentMemoryAdd,
		"mày nhớ gì về server chính":      IntentMemoryQuery,
		"chạy ls":                         IntentAction,
		"mày là gì":                       IntentChat,
		"":                                IntentUnknown,
	}
	for input, want := range tests {
		if got := Classify(input); got != want {
			t.Fatalf("Classify(%q) = %s, want %s", input, got, want)
		}
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

	r := NewModelRouter(cfg, database, providers.Enabled(cfg.Providers))
	_, err = r.Select(context.Background(), providers.ChatRequest{
		Messages:        []providers.Message{{Role: "user", Content: "hello"}},
		MaxOutputTokens: 512,
	})
	if err == nil {
		t.Fatal("expected approval error")
	}
}
