package providers_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vietclaw/internal/config"
	"vietclaw/internal/providers"
)

func TestMockProviderDeterministic(t *testing.T) {
	p := providers.New(config.ProviderConfig{ID: "mock", Type: "mock", Enabled: true, DefaultModel: "mock-small"})
	resp, err := p.Chat(context.Background(), providers.ChatRequest{
		Messages: []providers.Message{{Role: "user", Content: "mày là gì"}},
		Model:    "mock-small",
		Metadata: map[string]any{"language": "vi"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Provider != "mock" || resp.Model != "mock-small" || resp.Text == "" || resp.EstimatedCostUSD != 0 {
		t.Fatalf("unexpected mock response: %#v", resp)
	}
	if !strings.Contains(resp.Text, "VietClaw") {
		t.Fatalf("unexpected localized mock response: %q", resp.Text)
	}
}

func TestGeminiProviderChatAndStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, ":generateContent"):
			_, _ = fmt.Fprint(w, `{"candidates":[{"content":{"parts":[{"text":"xin chào"}]}}],"usageMetadata":{"promptTokenCount":3,"candidatesTokenCount":2}}`)
		case strings.Contains(r.URL.Path, ":streamGenerateContent"):
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = fmt.Fprint(w, "data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"xin\"}]}}]}\n\n")
			_, _ = fmt.Fprint(w, "data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\" chào\"}]}}]}\n\n")
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	p := providers.New(config.ProviderConfig{
		ID:           "gemini",
		Type:         providers.TypeGemini,
		Enabled:      true,
		DefaultModel: "gemini-test",
		BaseURL:      server.URL,
	})
	resp, err := p.Chat(context.Background(), providers.ChatRequest{
		Messages: []providers.Message{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Text != "xin chào" || resp.Provider != "gemini" {
		t.Fatalf("unexpected gemini response: %#v", resp)
	}

	ch, err := p.ChatStream(context.Background(), providers.ChatRequest{
		Messages: []providers.Message{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var streamed strings.Builder
	for chunk := range ch {
		if chunk.Error != "" {
			t.Fatal(chunk.Error)
		}
		streamed.WriteString(chunk.Text)
	}
	if streamed.String() != "xin chào" {
		t.Fatalf("streamed = %q", streamed.String())
	}
}

func TestAnthropicProviderRealStream(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = fmt.Fprint(w, "data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"xin\"}}\n\n")
		_, _ = fmt.Fprint(w, "data: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\" chào\"}}\n\n")
		_, _ = fmt.Fprint(w, "data: {\"type\":\"message_stop\"}\n\n")
	}))
	defer server.Close()

	p := providers.New(config.ProviderConfig{
		ID:           "anthropic",
		Type:         providers.TypeAnthropic,
		Enabled:      true,
		DefaultModel: "claude-test",
		BaseURL:      server.URL,
	})
	ch, err := p.ChatStream(context.Background(), providers.ChatRequest{
		Messages: []providers.Message{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var streamed strings.Builder
	for chunk := range ch {
		if chunk.Error != "" {
			t.Fatal(chunk.Error)
		}
		streamed.WriteString(chunk.Text)
	}
	if streamed.String() != "xin chào" {
		t.Fatalf("anthropic streamed = %q", streamed.String())
	}
}

func TestMockProviderEnglish(t *testing.T) {
	p := providers.New(config.ProviderConfig{ID: "mock", Type: "mock", Enabled: true, DefaultModel: "mock-small"})
	resp, err := p.Chat(context.Background(), providers.ChatRequest{
		Messages: []providers.Message{{Role: "user", Content: "what are you"}},
		Model:    "mock-small",
		Metadata: map[string]any{"language": "en"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(resp.Text, "lightweight agent runtime") {
		t.Fatalf("unexpected english mock response: %q", resp.Text)
	}
}
