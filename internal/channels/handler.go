package channels

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"vietclaw/internal/agent"
)

const defaultIdempotencyTTL = 10 * time.Minute

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
		Guard: NewTTLGuard(defaultIdempotencyTTL),
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
	if !h.Guard.Seen(msg.Platform + ":" + msg.MessageID) {
		return nil
	}
	if !ShouldHandle(msg, policy) {
		return nil
	}

	prompt := strings.TrimSpace(msg.Text)
	if prompt == "" {
		prompt = EmptyPromptReply
	}
	sessionID := SessionKey(msg)
	userID := UserIdentity(msg)

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
		reply = EmptyAgentReply
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
