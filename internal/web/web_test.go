package web

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"vietclaw/internal/agent"
	"vietclaw/internal/app"
	"vietclaw/internal/config"
	"vietclaw/internal/db"
	"vietclaw/internal/memory"
	"vietclaw/internal/version"
)

func TestAPIChatMemoryAddDoesNotCallProvider(t *testing.T) {
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
		ID:           "broken",
		Type:         "openai-compatible",
		Enabled:      true,
		DefaultModel: "broken-model",
		BaseURL:      "http://example.invalid",
		APIKeyEnv:    "VIETCLAW_TEST_MISSING_KEY",
	}}
	cfg.Router.DefaultProvider = "broken"
	cfg.Router.DefaultModel = "broken-model"

	application := &app.App{
		Config:    cfg,
		DB:        database,
		Logger:    log.New(bytes.NewBuffer(nil), "", 0),
		StartTime: time.Now(),
		Version:   version.Current(),
		Agent:     agent.NewService(cfg, database),
	}

	body := bytes.NewBufferString(`{"user_id":"local","channel":"web","message":"nhớ là server chính dùng Docker"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/chat", body)
	rec := httptest.NewRecorder()
	NewRouter(application).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	var resp agent.ChatResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if !resp.OK || resp.Provider != "local" || resp.Model != "rule" || resp.Intent != "memory_add" {
		t.Fatalf("unexpected response: %#v", resp)
	}
	if !strings.Contains(resp.Reply, "server chính dùng Docker") {
		t.Fatalf("reply did not include memory: %q", resp.Reply)
	}

	records, err := application.Agent.Memory().Search(context.Background(), "user:local", "Docker", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || records[0].Kind != memory.KindNote {
		t.Fatalf("memory not saved: %#v", records)
	}
}
