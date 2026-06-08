package agent

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"vietclaw/internal/config"
	contextbuilder "vietclaw/internal/context"
	"vietclaw/internal/memory"
	"vietclaw/internal/providers"
	"vietclaw/internal/router"
)

type ChatRequest struct {
	SessionID string `json:"session_id"`
	UserID    string `json:"user_id"`
	Channel   string `json:"channel"`
	Message   string `json:"message"`
	Mode      string `json:"mode"`
}

type ChatResponse struct {
	OK        bool    `json:"ok"`
	SessionID string  `json:"session_id"`
	Intent    string  `json:"intent"`
	Reply     string  `json:"reply"`
	Provider  string  `json:"provider"`
	Model     string  `json:"model"`
	CostUSD   float64 `json:"cost_usd"`
	Error     string  `json:"error,omitempty"`
}

type Service struct {
	cfg     config.Config
	db      *sql.DB
	mem     *memory.Store
	router  *router.ModelRouter
	context *contextbuilder.Builder
}

func NewService(cfg config.Config, db *sql.DB) *Service {
	mem := memory.NewStore(db)
	providerList := providers.Enabled(cfg.Providers)
	return &Service{
		cfg:     cfg,
		db:      db,
		mem:     mem,
		router:  router.NewModelRouter(cfg, db, providerList),
		context: contextbuilder.New(cfg, db, mem),
	}
}

func (s *Service) Memory() *memory.Store {
	return s.mem
}

func (s *Service) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	req = normalizeRequest(req, s.cfg)
	if strings.TrimSpace(req.Message) == "" {
		return ChatResponse{OK: false, SessionID: req.SessionID, Intent: string(router.IntentUnknown), Error: "message is required"}, fmt.Errorf("message is required")
	}

	if err := s.ensureSession(ctx, req); err != nil {
		return ChatResponse{}, err
	}
	if err := s.addMessage(ctx, req.SessionID, "user", req.Message); err != nil {
		return ChatResponse{}, err
	}

	intent := router.Classify(req.Message)
	runID := newID("run")
	if err := s.insertRun(ctx, runID, req.SessionID, string(intent), "", "", "running", ""); err != nil {
		return ChatResponse{}, err
	}

	switch intent {
	case router.IntentMemoryAdd:
		return s.handleMemoryAdd(ctx, req, runID, intent)
	case router.IntentMemoryQuery:
		return s.handleMemoryQuery(ctx, req, runID, intent)
	case router.IntentAction:
		reply := "tool action cần policy rõ hơn. shell.exec đang tắt mặc định nếu chưa bật trong config."
		_ = s.addMessage(ctx, req.SessionID, "assistant", reply)
		_ = s.finishRun(ctx, runID, "blocked", "tool policy blocked", "local", "rule")
		return ChatResponse{OK: true, SessionID: req.SessionID, Intent: string(intent), Reply: reply, Provider: "local", Model: "rule"}, nil
	default:
		return s.handleProviderChat(ctx, req, runID, intent)
	}
}

func (s *Service) handleMemoryAdd(ctx context.Context, req ChatRequest, runID string, intent router.Intent) (ChatResponse, error) {
	content := cleanMemoryContent(req.Message)
	rec, err := s.mem.Add(ctx, memory.Record{
		Scope:      "user:" + req.UserID,
		Kind:       memory.KindNote,
		Content:    content,
		Confidence: memory.ConfidenceConfirmed,
	})
	if err != nil {
		_ = s.finishRun(ctx, runID, "failed", err.Error(), "local", "rule")
		return ChatResponse{}, err
	}

	reply := "ok, t lưu: " + rec.Content
	_ = s.addMessage(ctx, req.SessionID, "assistant", reply)
	_ = s.finishRun(ctx, runID, "completed", reply, "local", "rule")
	return ChatResponse{OK: true, SessionID: req.SessionID, Intent: string(intent), Reply: reply, Provider: "local", Model: "rule"}, nil
}

func (s *Service) handleMemoryQuery(ctx context.Context, req ChatRequest, runID string, intent router.Intent) (ChatResponse, error) {
	query := cleanMemoryQuery(req.Message)
	records, err := s.mem.Search(ctx, "user:"+req.UserID, query, 5)
	if err != nil {
		_ = s.finishRun(ctx, runID, "failed", err.Error(), "local", "rule")
		return ChatResponse{}, err
	}

	reply := "t chưa thấy memory nào khớp."
	if len(records) > 0 {
		parts := make([]string, 0, len(records))
		for _, rec := range records {
			parts = append(parts, rec.Content)
		}
		reply = "t nhớ: " + strings.Join(parts, "; ")
	}
	_ = s.addMessage(ctx, req.SessionID, "assistant", reply)
	_ = s.finishRun(ctx, runID, "completed", reply, "local", "rule")
	return ChatResponse{OK: true, SessionID: req.SessionID, Intent: string(intent), Reply: reply, Provider: "local", Model: "rule"}, nil
}

func (s *Service) handleProviderChat(ctx context.Context, req ChatRequest, runID string, intent router.Intent) (ChatResponse, error) {
	messages, err := s.context.Messages(ctx, req.SessionID, req.UserID, req.Message)
	if err != nil {
		_ = s.finishRun(ctx, runID, "failed", err.Error(), "", "")
		return ChatResponse{}, err
	}

	chatReq := providers.ChatRequest{
		SessionID:       req.SessionID,
		Messages:        messages,
		Temperature:     0.2,
		MaxOutputTokens: 512,
		Metadata: map[string]any{
			"user_id": req.UserID,
			"channel": req.Channel,
			"mode":    req.Mode,
		},
	}
	selection, err := s.router.Select(ctx, chatReq)
	if err != nil {
		reply := err.Error()
		_ = s.addMessage(ctx, req.SessionID, "assistant", reply)
		_ = s.finishRun(ctx, runID, "needs_approval", reply, "", "")
		return ChatResponse{OK: false, SessionID: req.SessionID, Intent: string(intent), Reply: reply, Error: reply}, nil
	}
	chatReq.Model = selection.Model

	providerResp, err := selection.Provider.Chat(ctx, chatReq)
	if err != nil {
		_ = s.finishRun(ctx, runID, "failed", providerResp.RawError, providerResp.Provider, providerResp.Model)
		return ChatResponse{OK: false, SessionID: req.SessionID, Intent: string(intent), Provider: providerResp.Provider, Model: providerResp.Model, Error: providerResp.RawError}, err
	}

	_ = s.addMessage(ctx, req.SessionID, "assistant", providerResp.Text)
	_ = s.insertCost(ctx, providerResp)
	_ = s.finishRun(ctx, runID, "completed", providerResp.Text, providerResp.Provider, providerResp.Model)
	return ChatResponse{
		OK:        true,
		SessionID: req.SessionID,
		Intent:    string(intent),
		Reply:     providerResp.Text,
		Provider:  providerResp.Provider,
		Model:     providerResp.Model,
		CostUSD:   providerResp.EstimatedCostUSD,
	}, nil
}

func (s *Service) Sessions(ctx context.Context) ([]Session, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, channel, user_id, title, summary, created_at, updated_at FROM sessions ORDER BY updated_at DESC LIMIT 100`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	sessions := []Session{}
	for rows.Next() {
		var item Session
		if err := rows.Scan(&item.ID, &item.Channel, &item.UserID, &item.Title, &item.Summary, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		sessions = append(sessions, item)
	}
	return sessions, rows.Err()
}

func (s *Service) SessionMessages(ctx context.Context, id string) (SessionDetail, error) {
	var detail SessionDetail
	err := s.db.QueryRowContext(ctx, `SELECT id, channel, user_id, title, summary, created_at, updated_at FROM sessions WHERE id = ?`, id).
		Scan(&detail.Session.ID, &detail.Session.Channel, &detail.Session.UserID, &detail.Session.Title, &detail.Session.Summary, &detail.Session.CreatedAt, &detail.Session.UpdatedAt)
	if err != nil {
		return detail, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, session_id, role, content, created_at FROM messages WHERE session_id = ? ORDER BY id ASC`, id)
	if err != nil {
		return detail, err
	}
	defer rows.Close()
	for rows.Next() {
		var msg Message
		if err := rows.Scan(&msg.ID, &msg.SessionID, &msg.Role, &msg.Content, &msg.CreatedAt); err != nil {
			return detail, err
		}
		detail.Messages = append(detail.Messages, msg)
	}
	return detail, rows.Err()
}

type Session struct {
	ID        string         `json:"id"`
	Channel   string         `json:"channel"`
	UserID    string         `json:"user_id"`
	Title     sql.NullString `json:"title"`
	Summary   sql.NullString `json:"summary"`
	CreatedAt string         `json:"created_at"`
	UpdatedAt string         `json:"updated_at"`
}

type Message struct {
	ID        int64  `json:"id"`
	SessionID string `json:"session_id"`
	Role      string `json:"role"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

type SessionDetail struct {
	Session  Session   `json:"session"`
	Messages []Message `json:"messages"`
}

func (s *Service) ensureSession(ctx context.Context, req ChatRequest) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `
INSERT INTO sessions (id, channel, user_id, title, summary, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET updated_at = excluded.updated_at`,
		req.SessionID, req.Channel, req.UserID, sql.NullString{}, sql.NullString{}, now, now)
	return err
}

func (s *Service) addMessage(ctx context.Context, sessionID, role, content string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `INSERT INTO messages (session_id, role, content, created_at) VALUES (?, ?, ?, ?)`, sessionID, role, content, now)
	if err != nil {
		return err
	}
	_, _ = s.db.ExecContext(ctx, `UPDATE sessions SET updated_at = ? WHERE id = ?`, now, sessionID)
	return nil
}

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

func normalizeRequest(req ChatRequest, cfg config.Config) ChatRequest {
	if req.SessionID == "" {
		req.SessionID = newID("sess")
	}
	if req.UserID == "" {
		req.UserID = "local"
	}
	if req.Channel == "" {
		req.Channel = "web"
	}
	if req.Mode == "" {
		req.Mode = cfg.Agent.DefaultMode
	}
	return req
}

func cleanMemoryContent(message string) string {
	text := strings.TrimSpace(message)
	lower := strings.ToLower(text)
	prefixes := []string{"nhớ là", "lưu lại", "từ nay"}
	for _, prefix := range prefixes {
		if strings.HasPrefix(lower, prefix) {
			runes := []rune(text)
			return strings.TrimSpace(string(runes[len([]rune(prefix)):]))
		}
	}
	return text
}

func cleanMemoryQuery(message string) string {
	original := strings.TrimSpace(message)
	text := strings.ToLower(strings.TrimSpace(message))
	if strings.Contains(text, "server chính") {
		return "server chính"
	}
	replacers := []string{"mày nhớ gì về", "mày nhớ gì", "nhớ gì về", "server chính dùng gì", "server chính là gì", "?"}
	for _, item := range replacers {
		text = strings.ReplaceAll(text, item, "")
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return original
	}
	return text
}

func newID(prefix string) string {
	var data [8]byte
	if _, err := rand.Read(data[:]); err != nil {
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}
	return prefix + "_" + hex.EncodeToString(data[:])
}

func nullable(value string) sql.NullString {
	return sql.NullString{String: value, Valid: value != ""}
}
