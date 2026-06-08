package memory

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"vietclaw/internal/providers"
)

// SearchHybrid searches memories using both FTS/Keyword query and Vector Cosine Similarity
func (s *Store) SearchHybrid(ctx context.Context, scope, query string, limit int, embedder providers.Provider) ([]Record, error) {
	if limit <= 0 {
		limit = 10
	}

	// 1. Get initial candidates from database
	candidates, err := s.Search(ctx, scope, query, limit*3)
	if err != nil {
		return nil, err
	}

	// If no embedder or candidates, return standard list
	if embedder == nil || len(candidates) == 0 {
		if len(candidates) > limit {
			return candidates[:limit], nil
		}
		return candidates, nil
	}

	// 2. Generate embedding for query
	queryEmb, err := embedder.Embed(ctx, query)
	if err != nil {
		// Fallback to text search if embedding fails
		if len(candidates) > limit {
			return candidates[:limit], nil
		}
		return candidates, nil
	}

	// 3. Compute cosine similarity score and rank candidates
	type scoredRecord struct {
		record Record
		score  float32
	}
	var scoredList []scoredRecord

	for _, rec := range candidates {
		var similarity float32
		if len(rec.Embedding) > 0 {
			similarity = CosineSimilarity(queryEmb, rec.Embedding)
		} else {
			// fallback slight score if no embedding stored yet
			similarity = 0.1
		}
		scoredList = append(scoredList, scoredRecord{record: rec, score: similarity})
	}

	sort.Slice(scoredList, func(i, j int) bool {
		return scoredList[i].score > scoredList[j].score
	})

	var result []Record
	for i := 0; i < len(scoredList) && i < limit; i++ {
		result = append(result, scoredList[i].record)
	}

	return result, nil
}

func (s *Store) List(ctx context.Context, scope string, limit int) ([]Record, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	args := []any{}
	query := `SELECT id, scope, kind, content, confidence, created_at, updated_at, embedding FROM memories`
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
SELECT m.id, m.scope, m.kind, m.content, m.confidence, m.created_at, m.updated_at, m.embedding
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
SELECT id, scope, kind, content, confidence, created_at, updated_at, embedding
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

