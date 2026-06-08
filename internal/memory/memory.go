package memory

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type Kind string

const (
	KindProfile    Kind = "profile"
	KindPreference Kind = "preference"
	KindProject    Kind = "project"
	KindWorkflow   Kind = "workflow"
	KindDecision   Kind = "decision"
	KindConnection Kind = "connection"
	KindNote       Kind = "note"
)

type Confidence string

const (
	ConfidenceConfirmed Confidence = "confirmed"
	ConfidenceInferred  Confidence = "inferred"
	ConfidenceTemporary Confidence = "temporary"
)

type Record struct {
	ID         int64      `json:"id"`
	Scope      string     `json:"scope"`
	Kind       Kind       `json:"kind"`
	Content    string     `json:"content"`
	Confidence Confidence `json:"confidence"`
	CreatedAt  string     `json:"created_at"`
	UpdatedAt  string     `json:"updated_at"`
}

type Store struct {
	db     *sql.DB
	hasFTS bool
}

func NewStore(db *sql.DB) *Store {
	store := &Store{db: db}
	store.hasFTS = store.ensureFTS(context.Background()) == nil
	return store
}

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

	result, err := s.db.ExecContext(ctx, `
INSERT INTO memories (scope, kind, content, confidence, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?)`,
		rec.Scope, rec.Kind, rec.Content, confidenceValue(rec.Confidence), rec.CreatedAt, rec.UpdatedAt)
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

func (s *Store) List(ctx context.Context, scope string, limit int) ([]Record, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	args := []any{}
	query := `SELECT id, scope, kind, content, confidence, created_at, updated_at FROM memories`
	if scope != "" {
		query += ` WHERE scope = ?`
		args = append(args, scope)
	}
	query += ` ORDER BY id DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list memories: %w", err)
	}
	defer rows.Close()
	return scanRecords(rows)
}

func (s *Store) Search(ctx context.Context, scope, query string, limit int) ([]Record, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return s.List(ctx, scope, limit)
	}
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	if s.hasFTS {
		records, err := s.searchFTS(ctx, scope, query, limit)
		if err == nil {
			return records, nil
		}
	}
	return s.searchLike(ctx, scope, query, limit)
}

func (s *Store) searchFTS(ctx context.Context, scope, query string, limit int) ([]Record, error) {
	args := []any{query}
	sqlQuery := `
SELECT m.id, m.scope, m.kind, m.content, m.confidence, m.created_at, m.updated_at
FROM memories_fts f
JOIN memories m ON m.id = f.rowid
WHERE memories_fts MATCH ?`
	if scope != "" {
		sqlQuery += ` AND m.scope = ?`
		args = append(args, scope)
	}
	sqlQuery += ` ORDER BY m.id DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanRecords(rows)
}

func (s *Store) searchLike(ctx context.Context, scope, query string, limit int) ([]Record, error) {
	like := "%" + strings.ToLower(query) + "%"
	args := []any{like}
	sqlQuery := `
SELECT id, scope, kind, content, confidence, created_at, updated_at
FROM memories
WHERE lower(content) LIKE ?`
	if scope != "" {
		sqlQuery += ` AND scope = ?`
		args = append(args, scope)
	}
	sqlQuery += ` ORDER BY id DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("search memories: %w", err)
	}
	defer rows.Close()
	return scanRecords(rows)
}

func (s *Store) ensureFTS(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
CREATE VIRTUAL TABLE IF NOT EXISTS memories_fts USING fts5(scope, kind, content)`)
	return err
}

func scanRecords(rows *sql.Rows) ([]Record, error) {
	records := []Record{}
	for rows.Next() {
		var rec Record
		var kind string
		var confidence float64
		if err := rows.Scan(&rec.ID, &rec.Scope, &kind, &rec.Content, &confidence, &rec.CreatedAt, &rec.UpdatedAt); err != nil {
			return nil, err
		}
		rec.Kind = Kind(kind)
		rec.Confidence = confidenceLabel(confidence)
		records = append(records, rec)
	}
	return records, rows.Err()
}

func confidenceValue(conf Confidence) float64 {
	switch conf {
	case ConfidenceTemporary:
		return 0.35
	case ConfidenceInferred:
		return 0.7
	default:
		return 1.0
	}
}

func confidenceLabel(value float64) Confidence {
	if value < 0.5 {
		return ConfidenceTemporary
	}
	if value < 0.9 {
		return ConfidenceInferred
	}
	return ConfidenceConfirmed
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}
