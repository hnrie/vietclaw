package web_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"vietclaw/internal/agent"
	"vietclaw/internal/app"
	"vietclaw/internal/config"
	"vietclaw/internal/db"
	"vietclaw/internal/memory"
	"vietclaw/internal/version"
	"vietclaw/internal/web"
)

func TestAPIChatMemoryAddDoesNotCallProvider(t *testing.T) {
	application := testApp(t)
	cfg := application.Config
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
	application.Config = cfg
	application.Agent = agent.NewService(cfg, application.DB)

	body := bytes.NewBufferString(`{"user_id":"local","channel":"web","message":"nhớ là server chính dùng Docker"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/chat", body)
	rec := httptest.NewRecorder()
	web.NewRouter(application).ServeHTTP(rec, req)

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

func TestStaticFallbackServesAppRoutes(t *testing.T) {
	application := testApp(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/chat", nil)
	web.NewRouter(application).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "VietClaw") {
		t.Fatalf("expected embedded UI shell, got %q", rec.Body.String())
	}
}

func TestAPIRouteNotSwallowedByStaticFallback(t *testing.T) {
	application := testApp(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/does-not-exist", nil)
	web.NewRouter(application).ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d", rec.Code)
	}
}

func TestAPIChatStreamErrorIsJSONEvent(t *testing.T) {
	application := testApp(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/chat/stream", bytes.NewBufferString(`{"message":""}`))
	web.NewRouter(application).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	body := rec.Body.String()
	if !strings.Contains(body, "event: error") || !strings.Contains(body, `"error"`) {
		t.Fatalf("expected json error SSE event, got %q", body)
	}
}

func TestHarnessRunsAPI(t *testing.T) {
	application := testApp(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/harness/runs", bytes.NewBufferString(`{"goal":"fix failing auth test"}`))
	web.NewRouter(application).ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("create status = %d body = %s", rec.Code, rec.Body.String())
	}
	var created struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.ID == "" || created.Status == "" {
		t.Fatalf("unexpected create payload: %#v", created)
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/harness/runs/"+created.ID, nil)
	web.NewRouter(application).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("detail status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), "capsule.created") {
		t.Fatalf("expected evidence event in detail: %s", rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/harness/runs/"+created.ID+"/start", nil)
	web.NewRouter(application).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("start status = %d body = %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"blocked"`) {
		t.Fatalf("expected blocked non-git workspace: %s", rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/harness/runs/"+created.ID+"/diff", nil)
	web.NewRouter(application).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("diff status = %d body = %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/harness/runs/"+created.ID+"/cancel", nil)
	web.NewRouter(application).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("cancel status = %d body = %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodPost, "/api/harness/runs/"+created.ID+"/cleanup", nil)
	web.NewRouter(application).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("cleanup status = %d body = %s", rec.Code, rec.Body.String())
	}
}

func TestSettingsValidationRejectsInvalidConfig(t *testing.T) {
	application := testApp(t)
	cfg := application.Config
	cfg.Server.Port = -1
	body, _ := json.Marshal(cfg)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/settings", bytes.NewReader(body))
	web.NewRouter(application).ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
}

func TestMemoryDeleteAndCurate(t *testing.T) {
	application := testApp(t)
	ctx := context.Background()
	first, err := application.Agent.Memory().Add(ctx, memory.Record{Scope: "user:local", Content: "duplicate memory"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := application.Agent.Memory().Add(ctx, memory.Record{Scope: "user:local", Content: "duplicate   memory"}); err != nil {
		t.Fatal(err)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/memory/curate?scope=user:local", nil)
	web.NewRouter(application).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("curate status = %d body = %s", rec.Code, rec.Body.String())
	}

	rec = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodDelete, "/api/memory/"+strconv.FormatInt(first.ID, 10), nil)
	web.NewRouter(application).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("delete status = %d body = %s", rec.Code, rec.Body.String())
	}
}

func testApp(t *testing.T) *app.App {
	t.Helper()
	database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if err := db.ApplySchema(database); err != nil {
		t.Fatal(err)
	}

	cfg := config.Default(config.Paths{DataDir: t.TempDir()})
	return &app.App{
		Config:    cfg,
		DB:        database,
		Logger:    log.New(bytes.NewBuffer(nil), "", 0),
		StartTime: time.Now(),
		Version:   version.Current(),
		Agent:     agent.NewService(cfg, database),
	}
}
