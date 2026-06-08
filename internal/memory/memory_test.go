package memory

import (
	"context"
	"path/filepath"
	"testing"

	"vietclaw/internal/db"
)

func TestAddSearchList(t *testing.T) {
	database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if err := db.ApplySchema(database); err != nil {
		t.Fatal(err)
	}

	store := NewStore(database)
	ctx := context.Background()
	added, err := store.Add(ctx, Record{
		Scope:      "user:local",
		Kind:       KindPreference,
		Content:    "Minh thích tiết kiệm token",
		Confidence: ConfidenceConfirmed,
	})
	if err != nil {
		t.Fatal(err)
	}
	if added.ID == 0 {
		t.Fatal("missing inserted id")
	}

	results, err := store.Search(ctx, "user:local", "token", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Content != "Minh thích tiết kiệm token" {
		t.Fatalf("unexpected search results: %#v", results)
	}

	list, err := store.List(ctx, "user:local", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("list len = %d", len(list))
	}
}
