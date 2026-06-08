package memory

import (
	"context"
	"strings"
)

const duplicateSimilarityThreshold = 0.90

type CurationResult struct {
	Removed         int64             `json:"removed"`
	TextRemoved     int64             `json:"text_removed"`
	SemanticRemoved int64             `json:"semantic_removed"`
	Clusters        []CurationCluster `json:"clusters,omitempty"`
	Threshold       float32           `json:"threshold"`
}

type CurationCluster struct {
	KeptID     int64   `json:"kept_id"`
	RemovedID  int64   `json:"removed_id"`
	Similarity float32 `json:"similarity,omitempty"`
}

func (s *Store) CurateDuplicates(ctx context.Context, scope string) (CurationResult, error) {
	records, err := s.List(ctx, scope, 200)
	if err != nil {
		return CurationResult{}, err
	}
	seen := map[string]int64{}
	kept := []Record{}
	result := CurationResult{Threshold: duplicateSimilarityThreshold}
	for _, rec := range records {
		key := rec.Scope + "\x00" + strings.ToLower(strings.Join(strings.Fields(rec.Content), " "))
		if key == rec.Scope+"\x00" {
			continue
		}
		if keptID, ok := seen[key]; ok {
			if err := s.Delete(ctx, rec.ID); err != nil {
				return CurationResult{}, err
			}
			result.Removed++
			result.TextRemoved++
			result.Clusters = append(result.Clusters, CurationCluster{KeptID: keptID, RemovedID: rec.ID})
			continue
		}
		seen[key] = rec.ID

		if keptID, similarity := semanticDuplicate(rec, kept); keptID != 0 {
			if err := s.Delete(ctx, rec.ID); err != nil {
				return CurationResult{}, err
			}
			result.Removed++
			result.SemanticRemoved++
			result.Clusters = append(result.Clusters, CurationCluster{KeptID: keptID, RemovedID: rec.ID, Similarity: similarity})
			continue
		}
		kept = append(kept, rec)
	}
	return result, nil
}

func semanticDuplicate(rec Record, kept []Record) (int64, float32) {
	if len(rec.Embedding) == 0 {
		return 0, 0
	}
	for _, existing := range kept {
		if rec.Scope != existing.Scope || len(existing.Embedding) == 0 {
			continue
		}
		similarity := CosineSimilarity(rec.Embedding, existing.Embedding)
		if similarity >= duplicateSimilarityThreshold {
			return existing.ID, similarity
		}
	}
	return 0, 0
}
