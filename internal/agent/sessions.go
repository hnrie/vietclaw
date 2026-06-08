package agent

import (
	"context"
	"database/sql"
	"time"
)

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
