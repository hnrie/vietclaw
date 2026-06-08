package memory_test

import (
	"context"
	"path/filepath"
	"testing"

	"vietclaw/internal/db"
	"vietclaw/internal/memory"
)

func TestVectorConvert(t *testing.T) {
	slice := []float32{1.0, -2.5, 3.14}
	data := memory.Float32SliceToBytes(slice)
	restored := memory.BytesToFloat32Slice(data)

	if len(restored) != len(slice) {
		t.Fatalf("expected len %d, got %d", len(slice), len(restored))
	}
	for i := range slice {
		if slice[i] != restored[i] {
			t.Errorf("expected %f, got %f", slice[i], restored[i])
		}
	}
}

func TestCosineSimilarity(t *testing.T) {
	v1 := []float32{1.0, 0.0, 0.0}
	v2 := []float32{1.0, 0.0, 0.0}
	if sim := memory.CosineSimilarity(v1, v2); sim < 0.99 {
		t.Errorf("expected identity to be ~1.0, got %f", sim)
	}

	v3 := []float32{0.0, 1.0, 0.0}
	if sim := memory.CosineSimilarity(v1, v3); sim > 0.01 {
		t.Errorf("expected orthogonal vectors similarity to be ~0.0, got %f", sim)
	}
}

func TestHybridSearchFindsVectorOnlyCandidate(t *testing.T) {
	database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if err := db.ApplySchema(database); err != nil {
		t.Fatal(err)
	}

	store := memory.NewStore(database)
	if _, err := store.Add(context.Background(), memory.Record{
		Scope:      "user:local",
		Kind:       memory.KindNote,
		Content:    "deploy runbook lives in ops notebook",
		Confidence: memory.ConfidenceConfirmed,
		Embedding:  []float32{1, 0, 0},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Add(context.Background(), memory.Record{
		Scope:      "user:local",
		Kind:       memory.KindNote,
		Content:    "favorite color is blue",
		Confidence: memory.ConfidenceConfirmed,
		Embedding:  []float32{0, 1, 0},
	}); err != nil {
		t.Fatal(err)
	}

	results, err := store.SearchHybrid(context.Background(), "user:local", "release procedure", 1, fixedEmbedder{embedding: []float32{1, 0, 0}})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Content != "deploy runbook lives in ops notebook" {
		t.Fatalf("hybrid vector results = %#v", results)
	}
}
