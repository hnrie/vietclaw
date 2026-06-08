package providers

import (
	"context"
	"testing"

	"vietclaw/internal/config"
)

func TestMockProviderDeterministic(t *testing.T) {
	p := New(config.ProviderConfig{ID: "mock", Type: "mock", Enabled: true, DefaultModel: "mock-small"})
	resp, err := p.Chat(context.Background(), ChatRequest{
		Messages: []Message{{Role: "user", Content: "mày là gì"}},
		Model:    "mock-small",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Provider != "mock" || resp.Model != "mock-small" || resp.Text == "" || resp.EstimatedCostUSD != 0 {
		t.Fatalf("unexpected mock response: %#v", resp)
	}
}
