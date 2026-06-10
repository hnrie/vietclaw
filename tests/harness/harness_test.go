package harness_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"vietclaw/internal/config"
	"vietclaw/internal/db"
	"vietclaw/internal/harness"
)

func TestCreateHarnessRunStoresCapsuleAndEvents(t *testing.T) {
	database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if err := db.ApplySchema(database); err != nil {
		t.Fatal(err)
	}

	cfg := config.Default(config.Paths{DataDir: t.TempDir()})
	service := harness.New(cfg, database)
	run, err := service.Create(context.Background(), harness.CreateRequest{
		Goal: "fix failing auth test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if run.ID == "" || run.Status != harness.StatusPlanned {
		t.Fatalf("unexpected run: %#v", run)
	}
	if run.Mode != "agentless" || run.Risk != "low" {
		t.Fatalf("unexpected capsule defaults: %#v", run)
	}
	if len(run.Plan.Steps) == 0 {
		t.Fatalf("expected plan steps: %#v", run.Plan)
	}

	detail, err := service.Detail(context.Background(), run.ID)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Run.ID != run.ID {
		t.Fatalf("detail mismatch: %#v", detail)
	}
	if len(detail.Events) < 2 {
		t.Fatalf("expected ledger events, got %#v", detail.Events)
	}
}

func TestHarnessRiskGateMarksDangerousGoalsHigh(t *testing.T) {
	database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if err := db.ApplySchema(database); err != nil {
		t.Fatal(err)
	}

	cfg := config.Default(config.Paths{DataDir: t.TempDir()})
	service := harness.New(cfg, database)
	run, err := service.Create(context.Background(), harness.CreateRequest{
		Goal: "deploy to production and push git changes",
	})
	if err != nil {
		t.Fatal(err)
	}
	if run.Risk != "high" {
		t.Fatalf("risk = %q, want high", run.Risk)
	}
}

func TestHarnessRunAppliesPatchInGitWorktree(t *testing.T) {
	repo := tempGoRepo(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Messages []struct {
				Content string `json:"content"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		last := ""
		if len(req.Messages) > 0 {
			last = req.Messages[len(req.Messages)-1].Content
		}
		text := `{"summary":"plan","steps":{"localize":"read calc files","patch":"fix Add","verify":"go test ./..."},"stop_rules":["stop on forbidden tools"]}`
		if strings.Contains(last, "patcher") || strings.Contains(last, "unified diff") {
			text = strings.TrimSpace(`diff --git a/calc.go b/calc.go
--- a/calc.go
+++ b/calc.go
@@ -1,5 +1,5 @@
 package calc
 
 func Add(a, b int) int {
-	return a - b
+	return a + b
 }`)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"text": text, "provider": "fake", "model": "fake-model"})
	}))
	t.Cleanup(server.Close)

	database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if err := db.ApplySchema(database); err != nil {
		t.Fatal(err)
	}

	dataDir := t.TempDir()
	cfg := config.Default(config.Paths{DataDir: dataDir})
	cfg.Database.Path = filepath.Join(dataDir, "vietclaw.db")
	cfg.Agent.Workspace = repo
	cfg.Router.DefaultProvider = "fake"
	cfg.Router.DefaultModel = "fake-model"
	cfg.Providers = []config.ProviderConfig{{
		ID:           "fake",
		Type:         "http",
		Enabled:      true,
		DefaultModel: "fake-model",
		BaseURL:      server.URL,
	}}

	service := harness.New(cfg, database)
	run, err := service.Create(context.Background(), harness.CreateRequest{
		Goal:          "fix failing calc test",
		WorkspaceRoot: repo,
		AutoRun:       true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != harness.StatusPassed {
		detail, _ := service.Detail(context.Background(), run.ID)
		t.Fatalf("status = %s reason=%s detail=%#v events=%#v", run.Status, run.FailureReason, run, detail.Events)
	}
	if !strings.Contains(run.FinalDiff, "return a + b") {
		t.Fatalf("final diff missing patch: %s", run.FinalDiff)
	}
	if run.WorktreePath == "" {
		t.Fatalf("missing worktree path")
	}
	if out, err := exec.Command("git", "-C", repo, "diff", "--").CombinedOutput(); err != nil || strings.TrimSpace(string(out)) != "" {
		t.Fatalf("main repo changed or diff failed: err=%v out=%s", err, out)
	}
}

func TestHarnessRunBlocksNonGitWorkspace(t *testing.T) {
	workspace := t.TempDir()
	database, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if err := db.ApplySchema(database); err != nil {
		t.Fatal(err)
	}
	cfg := config.Default(config.Paths{DataDir: t.TempDir()})
	cfg.Agent.Workspace = workspace
	service := harness.New(cfg, database)
	run, err := service.Create(context.Background(), harness.CreateRequest{
		Goal:          "fix failing test",
		WorkspaceRoot: workspace,
		AutoRun:       true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if run.Status != harness.StatusBlocked {
		t.Fatalf("status = %s reason=%s", run.Status, run.FailureReason)
	}
}

func tempGoRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	writeFile(t, filepath.Join(repo, "go.mod"), "module example.com/calc\n\ngo 1.22\n")
	writeFile(t, filepath.Join(repo, "calc.go"), "package calc\n\nfunc Add(a, b int) int {\n\treturn a - b\n}\n")
	writeFile(t, filepath.Join(repo, "calc_test.go"), "package calc\n\nimport \"testing\"\n\nfunc TestAdd(t *testing.T) {\n\tif Add(2, 2) != 4 {\n\t\tt.Fatalf(\"bad add\")\n\t}\n}\n")
	runGit(t, repo, "init")
	runGit(t, repo, "config", "user.email", "test@example.com")
	runGit(t, repo, "config", "user.name", "Test User")
	runGit(t, repo, "add", ".")
	runGit(t, repo, "commit", "-m", "initial")
	return repo
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func runGit(t *testing.T, repo string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}
