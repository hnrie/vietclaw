package memory

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"vietclaw/internal/providers"
)

func (s *Store) SearchHybrid(ctx context.Context, scope, query string, limit int, embedder providers.Provider) ([]Record, error) {
	if limit <= 0 {
		limit = 10
	}

	keywordCandidates, err := s.Search(ctx, scope, query, limit*3)
	if err != nil {
		return nil, err
	}

	if embedder == nil || strings.TrimSpace(query) == "" {
		if len(keywordCandidates) > limit {
			return keywordCandidates[:limit], nil
		}
		return keywordCandidates, nil
	}

	queryEmb, err := embedder.Embed(ctx, query)
	if err != nil {
		if len(keywordCandidates) > limit {
			return keywordCandidates[:limit], nil
		}
		return keywordCandidates, nil
	}

	vectorCandidates, err := s.searchVectorCandidates(ctx, scope, queryEmb, limit*6)
	if err != nil {
		vectorCandidates = nil
	}

	type scoredRecord struct {
		record Record
		score  float32
	}

	scored := map[int64]scoredRecord{}
	for idx, rec := range keywordCandidates {
		score := float32(1.0)
		if len(rec.Embedding) > 0 {
			score += CosineSimilarity(queryEmb, rec.Embedding)
		}
		score += float32(len(keywordCandidates)-idx) * 0.001
		scored[rec.ID] = scoredRecord{record: rec, score: score}
	}
	for _, rec := range vectorCandidates {
		score := CosineSimilarity(queryEmb, rec.Embedding)
		if existing, ok := scored[rec.ID]; ok {
			if score > existing.score {
				existing.score = score
			}
			scored[rec.ID] = existing
			continue
		}
		scored[rec.ID] = scoredRecord{record: rec, score: score}
	}

	scoredList := make([]scoredRecord, 0, len(scored))
	for _, item := range scored {
		scoredList = append(scoredList, item)
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

func (s *Store) searchVectorCandidates(ctx context.Context, scope string, queryEmb []float32, limit int) ([]Record, error) {
	if len(queryEmb) == 0 {
		return nil, nil
	}
	if limit <= 0 {
		limit = 50
	}

	args := []any{}
	sqlQuery := `
SELECT id, scope, kind, content, confidence, created_at, updated_at, embedding
FROM memories
WHERE embedding IS NOT NULL`
	if scope != "" {
		sqlQuery += ` AND scope = ?`
		args = append(args, scope)
	}
	sqlQuery += ` ORDER BY id DESC LIMIT ?`
	args = append(args, limit)

	rows, err := s.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	records, err := scanRecords(rows)
	if err != nil {
		return nil, err
	}
	sort.Slice(records, func(i, j int) bool {
		return CosineSimilarity(queryEmb, records[i].Embedding) > CosineSimilarity(queryEmb, records[j].Embedding)
	})
	return records, nil
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
