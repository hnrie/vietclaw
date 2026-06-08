package channels

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"vietclaw/internal/agent"
	"vietclaw/internal/config"
)

type Adapter interface {
	Name() string
	Start(ctx context.Context) error
}

type Sender func(ctx context.Context, msg InboundMessage, reply string) error

type InboundMessage struct {
	Platform     string
	MessageID    string
	GuildID      string
	ChannelID    string
	ThreadID     string
	ChatID       string
	UserID       string
	Username     string
	IsDM         bool
	IsGroup      bool
	IsReplyToBot bool
	MentionsBot  bool
	Text         string
	RawText      string
	CreatedAt    time.Time
}

type Policy struct {
	RespondInDM     bool
	RespondInGroups string
}

type Status struct {
	Name    string `json:"name"`
	Enabled bool   `json:"enabled"`
	Running bool   `json:"running"`
	Error   string `json:"error,omitempty"`
}

type Handler struct {
	Agent *agent.Service
	DB    *sql.DB
	Log   *log.Logger
	Guard *TTLGuard
}

func NewHandler(service *agent.Service, db *sql.DB, logger *log.Logger) *Handler {
	return &Handler{
		Agent: service,
		DB:    db,
		Log:   logger,
		Guard: NewTTLGuard(10 * time.Minute),
	}
}

func (h *Handler) Handle(ctx context.Context, msg InboundMessage, policy Policy, botMentions []string, send Sender) error {
	msg.Text = StripMentions(msg.Text, botMentions)
	msg.RawText = defaultString(msg.RawText, msg.Text)
	if msg.CreatedAt.IsZero() {
		msg.CreatedAt = time.Now().UTC()
	}
	if msg.MessageID == "" || msg.UserID == "" {
		return fmt.Errorf("message_id and user_id are required")
	}
	key := msg.Platform + ":" + msg.MessageID
	if !h.Guard.Seen(key) {
		return nil
	}
	if !ShouldHandle(msg, policy) {
		return nil
	}

	prompt := strings.TrimSpace(msg.Text)
	if prompt == "" {
		prompt = "gọi t rồi muốn t làm gì?"
	}
	sessionID := SessionKey(msg)
	userID := msg.Platform + ":" + msg.UserID
	if msg.IsGroup {
		userID = msg.Platform + ":group:" + defaultString(msg.GuildID, msg.ChatID) + ":user:" + msg.UserID
	}

	if err := h.insertChannelMessage(ctx, msg, sessionID, userID, "in", prompt); err != nil && h.Log != nil {
		h.Log.Printf("channel message log failed platform=%s direction=in err=%v", msg.Platform, err)
	}
	resp, err := h.Agent.Chat(ctx, agent.ChatRequest{
		SessionID: sessionID,
		UserID:    userID,
		Channel:   msg.Platform,
		Message:   prompt,
	})
	if err != nil {
		if h.Log != nil {
			h.Log.Printf("channel agent failed platform=%s err=%v", msg.Platform, err)
		}
		return err
	}
	reply := strings.TrimSpace(resp.Reply)
	if reply == "" {
		reply = "t chưa có gì để trả lời."
	}
	if err := send(ctx, msg, reply); err != nil {
		if h.Log != nil {
			h.Log.Printf("channel send failed platform=%s err=%v", msg.Platform, err)
		}
		return err
	}
	_ = h.insertChannelMessage(ctx, msg, sessionID, userID, "out", reply)
	if h.Log != nil {
		h.Log.Printf("channel agent success platform=%s session=%s intent=%s", msg.Platform, sessionID, resp.Intent)
	}
	return nil
}

func ShouldHandle(msg InboundMessage, policy Policy) bool {
	if msg.IsDM {
		return policy.RespondInDM
	}
	return msg.MentionsBot || msg.IsReplyToBot
}

func StripMentions(text string, mentions []string) string {
	cleaned := strings.TrimSpace(text)
	for {
		changed := false
		for _, mention := range mentions {
			if mention == "" {
				continue
			}
			if strings.HasPrefix(strings.ToLower(cleaned), strings.ToLower(mention)) {
				cleaned = strings.TrimSpace(cleaned[len(mention):])
				changed = true
			}
		}
		if !changed {
			return cleaned
		}
	}
}

func SessionKey(msg InboundMessage) string {
	switch msg.Platform {
	case "discord":
		if msg.IsDM {
			return "discord:dm:" + msg.UserID
		}
		if msg.ThreadID != "" {
			return "discord:guild:" + msg.GuildID + ":thread:" + msg.ThreadID
		}
		return "discord:guild:" + msg.GuildID + ":channel:" + msg.ChannelID
	case "telegram":
		if msg.IsDM {
			return "telegram:private:" + msg.UserID
		}
		if msg.ThreadID != "" {
			return "telegram:chat:" + msg.ChatID + ":topic:" + msg.ThreadID
		}
		return "telegram:chat:" + msg.ChatID
	default:
		if msg.IsDM {
			return msg.Platform + ":dm:" + msg.UserID
		}
		return msg.Platform + ":chat:" + defaultString(msg.ChatID, msg.ChannelID)
	}
}

func (h *Handler) insertChannelMessage(ctx context.Context, msg InboundMessage, sessionID, userID, direction, content string) error {
	if h.DB == nil {
		return nil
	}
	_, err := h.DB.ExecContext(ctx, `
INSERT OR IGNORE INTO channel_messages (platform, message_id, session_id, user_id, direction, content, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?)`,
		msg.Platform, msg.MessageID, sessionID, userID, direction, content, time.Now().UTC().Format(time.RFC3339))
	return err
}

type TTLGuard struct {
	ttl   time.Duration
	mu    sync.Mutex
	items map[string]time.Time
}

func NewTTLGuard(ttl time.Duration) *TTLGuard {
	return &TTLGuard{ttl: ttl, items: map[string]time.Time{}}
}

func (g *TTLGuard) Seen(key string) bool {
	now := time.Now()
	g.mu.Lock()
	defer g.mu.Unlock()
	for item, expires := range g.items {
		if now.After(expires) {
			delete(g.items, item)
		}
	}
	if expires, ok := g.items[key]; ok && now.Before(expires) {
		return false
	}
	g.items[key] = now.Add(g.ttl)
	return true
}

type Manager struct {
	cfg      config.Config
	adapters []Adapter
	logger   *log.Logger
	statuses map[string]Status
	mu       sync.Mutex
}

func NewManager(cfg config.Config, logger *log.Logger, adapters []Adapter) *Manager {
	statuses := map[string]Status{
		"discord":  {Name: "discord", Enabled: cfg.Channels.Discord.Enabled},
		"telegram": {Name: "telegram", Enabled: cfg.Channels.Telegram.Enabled},
	}
	return &Manager{cfg: cfg, logger: logger, adapters: adapters, statuses: statuses}
}

func (m *Manager) Start(ctx context.Context) {
	for _, adapter := range m.adapters {
		adapter := adapter
		m.setRunning(adapter.Name(), true, "")
		go func() {
			if m.logger != nil {
				m.logger.Printf("channel adapter starting name=%s", adapter.Name())
			}
			err := adapter.Start(ctx)
			if err != nil && ctx.Err() == nil {
				m.setRunning(adapter.Name(), false, err.Error())
				if m.logger != nil {
					m.logger.Printf("channel adapter failed name=%s err=%v", adapter.Name(), err)
				}
				return
			}
			m.setRunning(adapter.Name(), false, "")
			if m.logger != nil {
				m.logger.Printf("channel adapter stopped name=%s", adapter.Name())
			}
		}()
	}
}

func (m *Manager) Statuses() []Status {
	m.mu.Lock()
	defer m.mu.Unlock()
	return []Status{m.statuses["discord"], m.statuses["telegram"]}
}

func (m *Manager) setRunning(name string, running bool, errText string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	status := m.statuses[name]
	status.Name = name
	status.Running = running
	status.Error = errText
	m.statuses[name] = status
}

func StatusFromConfig(cfg config.Config) []Status {
	return []Status{
		{Name: "discord", Enabled: cfg.Channels.Discord.Enabled},
		{Name: "telegram", Enabled: cfg.Channels.Telegram.Enabled},
	}
}

func DiscordPolicy(cfg config.DiscordConfig) Policy {
	return Policy{RespondInDM: cfg.RespondInDM, RespondInGroups: cfg.RespondInGuilds}
}

func TelegramPolicy(cfg config.TelegramConfig) Policy {
	return Policy{RespondInDM: cfg.RespondInPrivate, RespondInGroups: cfg.RespondInGroups}
}

func Allowed(value string, allowed []string) bool {
	if len(allowed) == 0 {
		return true
	}
	for _, item := range allowed {
		if value == item {
			return true
		}
	}
	return false
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
