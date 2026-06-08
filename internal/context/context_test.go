package contextbuilder

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"vietclaw/internal/config"
	"vietclaw/internal/db"
	"vietclaw/internal/memory"
)

func TestBuilderLimitsContextChars(t *testing.T) {
	database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if err := db.ApplySchema(database); err != nil {
		t.Fatal(err)
	}

	cfg := config.Default(config.Paths{DataDir: t.TempDir()})
	cfg.Agent.MaxContextChars = 120
	store := memory.NewStore(database)
	if _, err := store.Add(context.Background(), memory.Record{
		Scope:   "user:local",
		Kind:    memory.KindNote,
		Content: strings.Repeat("token ", 200),
	}); err != nil {
		t.Fatal(err)
	}

	builder := New(cfg, database, store)
	messages, err := builder.Messages(context.Background(), "", "local", "token")
	if err != nil {
		t.Fatal(err)
	}
	if len([]rune(messages[0].Content)) > 120 {
		t.Fatalf("context len = %d", len([]rune(messages[0].Content)))
	}
}
