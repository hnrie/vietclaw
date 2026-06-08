package memory

import (
	"context"
	"fmt"
	"strings"
	"time"
)

func (s *Store) Add(ctx context.Context, rec Record) (Record, error) {
	rec.Scope = defaultString(rec.Scope, "user:local")
	if rec.Kind == "" {
		rec.Kind = KindNote
	}
	if rec.Confidence == "" {
		rec.Confidence = ConfidenceConfirmed
	}
	rec.Content = strings.TrimSpace(rec.Content)
	if rec.Content == "" {
		return Record{}, fmt.Errorf("memory content is required")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	rec.CreatedAt = now
	rec.UpdatedAt = now

	var embBytes []byte
	if len(rec.Embedding) > 0 {
		embBytes = Float32SliceToBytes(rec.Embedding)
	}

	result, err := s.db.ExecContext(ctx, `
INSERT INTO memories (scope, kind, content, confidence, created_at, updated_at, embedding)
VALUES (?, ?, ?, ?, ?, ?, ?)`,
		rec.Scope, rec.Kind, rec.Content, confidenceValue(rec.Confidence), rec.CreatedAt, rec.UpdatedAt, embBytes)
	if err != nil {
		return Record{}, fmt.Errorf("add memory: %w", err)
	}
	rec.ID, _ = result.LastInsertId()

	if s.hasFTS {
		_, _ = s.db.ExecContext(ctx, `INSERT INTO memories_fts(rowid, scope, kind, content) VALUES (?, ?, ?, ?)`,
			rec.ID, rec.Scope, rec.Kind, rec.Content)
	}
	return rec, nil
}
