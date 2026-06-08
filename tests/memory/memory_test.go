package memory_test

import (
	"context"
	"path/filepath"
	"testing"

	"vietclaw/internal/db"
	"vietclaw/internal/memory"
)

func TestStoreAddAndSearch(t *testing.T) {
	database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if err := db.ApplySchema(database); err != nil {
		t.Fatal(err)
	}

	store := memory.NewStore(database)
	rec, err := store.Add(context.Background(), memory.Record{
		Scope:      "user:local",
		Kind:       memory.KindNote,
		Content:    "Minh thích tiết kiệm token",
		Confidence: memory.ConfidenceConfirmed,
	})
	if err != nil {
		t.Fatal(err)
	}
	if rec.ID == 0 {
		t.Fatal("expected persisted id")
	}

	results, err := store.Search(context.Background(), "user:local", "token", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Content != "Minh thích tiết kiệm token" {
		t.Fatalf("search results = %#v", results)
	}
}

func TestCurateDuplicatesUsesSemanticSimilarity(t *testing.T) {
	database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if err := db.ApplySchema(database); err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	store := memory.NewStore(database)
	_, err = store.Add(ctx, memory.Record{
		Scope:     "user:local",
		Content:   "primary memory",
		Embedding: []float32{1, 0, 0},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.Add(ctx, memory.Record{
		Scope:     "user:local",
		Content:   "different words same meaning",
		Embedding: []float32{0.99, 0.01, 0},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = store.Add(ctx, memory.Record{
		Scope:     "agent:other",
		Content:   "other scope similar",
		Embedding: []float32{1, 0, 0},
	})
	if err != nil {
		t.Fatal(err)
	}

	result, err := store.CurateDuplicates(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if result.SemanticRemoved != 1 || result.Removed != 1 {
		t.Fatalf("curation result = %#v", result)
	}
	remaining, err := store.List(ctx, "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(remaining) != 2 {
		t.Fatalf("remaining memories = %#v", remaining)
	}
}
