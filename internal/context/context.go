package contextbuilder

import (
	"context"
	"database/sql"
	"strings"

	"vietclaw/internal/config"
	"vietclaw/internal/memory"
	"vietclaw/internal/providers"
)

type Builder struct {
	cfg config.Config
	db  *sql.DB
	mem *memory.Store
}

func New(cfg config.Config, db *sql.DB, mem *memory.Store) *Builder {
	return &Builder{cfg: cfg, db: db, mem: mem}
}

func (b *Builder) Messages(ctx context.Context, sessionID, userID, userMessage string) ([]providers.Message, error) {
	maxChars := b.cfg.Agent.MaxContextChars
	if maxChars <= 0 {
		maxChars = 24000
	}

	var parts []string
	parts = append(parts, "Bạn là VietClaw, agent điều phối nhẹ. Trả lời tiếng Việt ngắn, tự nhiên.")

	scope := "user:" + userID
	memories, _ := b.mem.Search(ctx, scope, userMessage, 6)
	if len(memories) > 0 {
		lines := []string{"Memory liên quan:"}
		for _, rec := range memories {
			lines = append(lines, "- "+rec.Content)
		}
		parts = append(parts, strings.Join(lines, "\n"))
	}

	history := b.history(ctx, sessionID)
	if history != "" {
		parts = append(parts, "Lịch sử gần đây:\n"+history)
	}

	system := trimTo(strings.Join(parts, "\n\n"), maxChars)
	return []providers.Message{
		{Role: "system", Content: system},
		{Role: "user", Content: userMessage},
	}, nil
}

func (b *Builder) history(ctx context.Context, sessionID string) string {
	if sessionID == "" || b.db == nil {
		return ""
	}
	limit := b.cfg.Agent.MaxHistoryMessages
	if limit <= 0 {
		limit = 12
	}
	rows, err := b.db.QueryContext(ctx, `
SELECT role, content FROM messages
WHERE session_id = ?
ORDER BY id DESC
LIMIT ?`, sessionID, limit)
	if err != nil {
		return ""
	}
	defer rows.Close()

	var lines []string
	for rows.Next() {
		var role, content string
		if rows.Scan(&role, &content) == nil {
			lines = append(lines, role+": "+content)
		}
	}
	for i, j := 0, len(lines)-1; i < j; i, j = i+1, j-1 {
		lines[i], lines[j] = lines[j], lines[i]
	}
	return trimTo(strings.Join(lines, "\n"), b.cfg.Agent.MaxContextChars/2)
}

func trimTo(value string, max int) string {
	if max <= 0 || len([]rune(value)) <= max {
		return value
	}
	runes := []rune(value)
	return string(runes[len(runes)-max:])
}
