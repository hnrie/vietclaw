package agent_test

import (
	"context"
	"path/filepath"
	"testing"

	"vietclaw/internal/agent"
	"vietclaw/internal/config"
	"vietclaw/internal/db"
)

func TestAgentProfileMemoryIsolation(t *testing.T) {
	service, cleanup := testService(t)
	defer cleanup()

	ctx := context.Background()
	resp, err := service.Chat(ctx, agent.ChatRequest{
		UserID:  "local",
		AgentID: "researcher",
		Message: "nhớ là researcher dùng notebook riêng",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.AgentID != "researcher" {
		t.Fatalf("agent id = %q", resp.AgentID)
	}

	researcher, err := service.Memory().Search(ctx, "agent:researcher:user:local", "notebook", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(researcher) != 1 {
		t.Fatalf("researcher memory = %#v", researcher)
	}
	defaultScope, err := service.Memory().Search(ctx, "user:local", "notebook", 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(defaultScope) != 0 {
		t.Fatalf("default memory should be isolated: %#v", defaultScope)
	}
}

func TestDelegateMentionSelectsAgentProfile(t *testing.T) {
	service, cleanup := testService(t)
	defer cleanup()

	resp, err := service.Chat(context.Background(), agent.ChatRequest{
		UserID:  "local",
		Message: "@researcher nhớ là delegated memory",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.AgentID != "researcher" {
		t.Fatalf("delegated agent id = %q", resp.AgentID)
	}
}

func TestLLMDelegateSelectsAgentProfile(t *testing.T) {
	service, cleanup := testService(t)
	defer cleanup()

	resp, err := service.Chat(context.Background(), agent.ChatRequest{
		UserID:  "local",
		Message: "please research storage options for this project",
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.AgentID != "researcher" {
		t.Fatalf("llm delegated agent id = %q", resp.AgentID)
	}
}

func testService(t *testing.T) (*agent.Service, func()) {
	t.Helper()
	database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.ApplySchema(database); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default(config.Paths{DataDir: t.TempDir()})
	cfg.Agents = append(cfg.Agents, config.AgentProfileConfig{
		ID:      "researcher",
		Name:    "Researcher",
		Persona: "Focus on research tasks.",
	})
	return agent.NewService(cfg, database), func() { _ = database.Close() }
}
