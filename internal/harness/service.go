package harness

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"vietclaw/internal/agent"
	"vietclaw/internal/config"
)

const (
	defaultMode       = "agentless"
	defaultRisk       = "medium"
	defaultMaxTokens  = 24000
	defaultMaxUSD     = 0.25
	defaultMaxMinutes = 20
)

type Service struct {
	cfg config.Config
	db  *sql.DB
}

func New(cfg config.Config, db *sql.DB) *Service {
	return &Service{cfg: cfg, db: db}
}

func (s *Service) harnessConfig() config.Config {
	cfg := s.cfg
	if cfg.Agent.MaxOutputTokens < 2048 {
		cfg.Agent.MaxOutputTokens = 2048
	}
	return cfg
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (Capsule, error) {
	req = normalizeCreateRequest(req)
	runID := newID("harness")
	now := time.Now().UTC().Format(time.RFC3339)
	capsule := Capsule{
		ID:             runID,
		SessionID:      req.SessionID,
		Goal:           req.Goal,
		Mode:           req.Mode,
		Risk:           req.Risk,
		Status:         StatusPlanned,
		Budget:         Budget{MaxTokens: req.MaxTokens, MaxUSD: req.MaxUSD, MaxMinutes: req.MaxMinutes},
		AllowedTools:   req.AllowedTools,
		ForbiddenTools: req.ForbiddenTools,
		SuccessChecks:  req.SuccessChecks,
		WorkspaceRoot:  req.WorkspaceRoot,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := s.insertRun(ctx, capsule); err != nil {
		return Capsule{}, err
	}
	_ = s.addEvent(ctx, runID, "capsule.created", mustJSON(map[string]any{
		"goal":            capsule.Goal,
		"mode":            capsule.Mode,
		"risk":            capsule.Risk,
		"allowed_tools":   capsule.AllowedTools,
		"forbidden_tools": capsule.ForbiddenTools,
		"success_checks":  capsule.SuccessChecks,
	}))

	plan, providerID, model, err := s.planWithProvider(ctx, capsule)
	if err != nil {
		capsule.Status = StatusNeedsApproval
		capsule.Summary = err.Error()
		capsule.Plan = fallbackPlan(capsule, err.Error())
		_ = s.updateRunPlan(ctx, capsule)
		_ = s.addEvent(ctx, runID, "plan.failed", mustJSON(map[string]any{"error": err.Error()}))
		return capsule, nil
	}

	capsule.Provider = providerID
	capsule.Model = model
	capsule.Plan = plan
	capsule.Summary = plan.Summary
	if err := s.updateRunPlan(ctx, capsule); err != nil {
		return Capsule{}, err
	}
	_ = s.addEvent(ctx, runID, "plan.created", mustJSON(map[string]any{
		"provider": providerID,
		"model":    model,
	}))
	if req.AutoRun {
		return s.Start(ctx, runID)
	}
	return capsule, nil
}

func (s *Service) List(ctx context.Context, limit int) ([]Capsule, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.db.QueryContext(ctx, `
SELECT id, session_id, goal, mode, risk, status, budget_json, allowed_tools_json,
       forbidden_tools_json, success_checks_json, provider, model, summary, plan_json,
       workspace_root, worktree_path, branch_name, base_ref, changed_files_json, final_diff, failure_reason,
       created_at, updated_at
FROM harness_runs
ORDER BY created_at DESC
LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Capsule
	for rows.Next() {
		capsule, err := scanCapsule(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, capsule)
	}
	return out, rows.Err()
}

func (s *Service) Detail(ctx context.Context, id string) (RunDetail, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, session_id, goal, mode, risk, status, budget_json, allowed_tools_json,
       forbidden_tools_json, success_checks_json, provider, model, summary, plan_json,
       workspace_root, worktree_path, branch_name, base_ref, changed_files_json, final_diff, failure_reason,
       created_at, updated_at
FROM harness_runs
WHERE id = ?`, id)
	capsule, err := scanCapsule(row)
	if err != nil {
		return RunDetail{}, err
	}
	events, err := s.events(ctx, id)
	if err != nil {
		return RunDetail{}, err
	}
	return RunDetail{Run: capsule, Events: events}, nil
}

func (s *Service) Start(ctx context.Context, id string) (Capsule, error) {
	capsule, err := s.load(ctx, id)
	if err != nil {
		return Capsule{}, err
	}
	if capsule.Status == StatusPassed || capsule.Status == StatusCancelled {
		return capsule, nil
	}
	if err := s.updateStatus(ctx, id, StatusRunning, ""); err != nil {
		return Capsule{}, err
	}
	capsule.Status = StatusRunning

	root, err := resolveWorkspaceRoot(capsule.WorkspaceRoot, s.cfg)
	if err != nil {
		return s.block(ctx, capsule, err.Error())
	}
	gitRoot, err := gitTopLevel(ctx, root)
	if err != nil {
		return s.block(ctx, capsule, "workspace is not a git repo: "+root)
	}
	capsule.WorkspaceRoot = gitRoot
	baseRef, _ := runCommand(ctx, gitRoot, []string{"git", "rev-parse", "HEAD"})
	capsule.BaseRef = strings.TrimSpace(baseRef.Output)
	capsule.BranchName = "vietclaw/harness/" + capsule.ID
	capsule.WorktreePath = filepath.Join(s.dataDir(), "harness-worktrees", capsule.ID)
	if err := s.updateExecution(ctx, capsule); err != nil {
		return Capsule{}, err
	}

	if err := s.prepareWorktree(ctx, &capsule); err != nil {
		return s.block(ctx, capsule, err.Error())
	}
	defer func() {
		if final, err := s.load(context.Background(), id); err == nil && final.Status == StatusRunning {
			_ = s.updateStatus(context.Background(), id, StatusFailed, "run stopped before terminal status")
		}
	}()

	contextPack := s.contextPack(ctx, capsule)
	_ = s.addEvent(ctx, capsule.ID, "context.pack", mustJSON(contextPack))

	var lastVerify commandResult
	for attempt := 1; attempt <= 3; attempt++ {
		if s.cancelled(ctx, capsule.ID) {
			capsule.Status = StatusCancelled
			return capsule, nil
		}
		if err := s.updateStatus(ctx, capsule.ID, StatusLocalizing, ""); err != nil {
			return Capsule{}, err
		}
		localized := s.localize(ctx, capsule, contextPack, attempt)
		_ = s.addEvent(ctx, capsule.ID, "step.localize", mustJSON(localized))

		if err := s.updateStatus(ctx, capsule.ID, StatusPatching, ""); err != nil {
			return Capsule{}, err
		}
		patchText, providerID, model, err := s.patchWithProvider(ctx, capsule, contextPack, localized, lastVerify, attempt)
		capsule.Provider = defaultString(providerID, capsule.Provider)
		capsule.Model = defaultString(model, capsule.Model)
		_ = s.updateExecution(ctx, capsule)
		if err != nil {
			return s.fail(ctx, capsule, err.Error())
		}
		if blocked := forbiddenText(patchText); blocked != "" {
			return s.block(ctx, capsule, blocked)
		}
		applied, err := s.applyPatch(ctx, capsule, patchText, attempt)
		if err != nil {
			_ = s.addEvent(ctx, capsule.ID, "patch.apply_failed", mustJSON(map[string]any{"attempt": attempt, "error": err.Error(), "preview": truncate(patchText, 1200)}))
			lastVerify = commandResult{Command: "git apply", ExitCode: 1, Output: err.Error()}
			continue
		}
		_ = s.addEvent(ctx, capsule.ID, "patch.applied", mustJSON(map[string]any{"attempt": attempt, "changed": applied}))

		if err := s.updateStatus(ctx, capsule.ID, StatusVerifying, ""); err != nil {
			return Capsule{}, err
		}
		verify := s.verify(ctx, capsule, localized)
		lastVerify = verify
		_ = s.addEvent(ctx, capsule.ID, "verify.result", mustJSON(verify))
		if verify.ExitCode == 0 {
			diff := s.finalDiff(ctx, capsule)
			changed := s.changedFiles(ctx, capsule)
			capsule.FinalDiff = diff
			capsule.ChangedFiles = changed
			capsule.Summary = "harness passed: patch verified"
			capsule.Status = StatusPassed
			if err := s.updateExecution(ctx, capsule); err != nil {
				return Capsule{}, err
			}
			_ = s.updateStatus(ctx, capsule.ID, StatusPassed, "")
			_ = s.addEvent(ctx, capsule.ID, "run.passed", mustJSON(map[string]any{"changed_files": changed, "diff_len": len(diff)}))
			return s.load(ctx, capsule.ID)
		}
	}
	return s.fail(ctx, capsule, "verification failed after 3 attempts")
}

func (s *Service) Cancel(ctx context.Context, id string) (Capsule, error) {
	if err := s.updateStatus(ctx, id, StatusCancelled, "cancelled by user"); err != nil {
		return Capsule{}, err
	}
	_ = s.addEvent(ctx, id, "run.cancelled", "{}")
	return s.load(ctx, id)
}

func (s *Service) Diff(ctx context.Context, id string) (string, error) {
	capsule, err := s.load(ctx, id)
	if err != nil {
		return "", err
	}
	if capsule.FinalDiff != "" {
		return capsule.FinalDiff, nil
	}
	if capsule.WorktreePath == "" {
		return "", nil
	}
	return s.finalDiff(ctx, capsule), nil
}

func (s *Service) Cleanup(ctx context.Context, id string) (Capsule, error) {
	capsule, err := s.load(ctx, id)
	if err != nil {
		return Capsule{}, err
	}
	if capsule.WorktreePath != "" {
		_, _ = runCommand(ctx, capsule.WorkspaceRoot, []string{"git", "worktree", "remove", "--force", capsule.WorktreePath})
		_ = os.RemoveAll(capsule.WorktreePath)
	}
	if strings.HasPrefix(capsule.BranchName, "vietclaw/harness/") && capsule.WorkspaceRoot != "" {
		_, _ = runCommand(ctx, capsule.WorkspaceRoot, []string{"git", "branch", "-D", capsule.BranchName})
	}
	_ = s.addEvent(ctx, capsule.ID, "worktree.cleaned", mustJSON(map[string]any{"worktree_path": capsule.WorktreePath, "branch_name": capsule.BranchName}))
	capsule.WorktreePath = ""
	capsule.BranchName = ""
	if err := s.updateExecution(ctx, capsule); err != nil {
		return Capsule{}, err
	}
	return s.load(ctx, id)
}

func (s *Service) planWithProvider(ctx context.Context, capsule Capsule) (Plan, string, string, error) {
	prompt := "Lập kế hoạch JSON cho VietClaw Harness. Chỉ trả JSON có summary, steps, stop_rules. " +
		"Steps gồm 3 mục localize, patch, verify. " +
		"Goal: " + capsule.Goal + ". Mode: " + capsule.Mode + ". Risk: " + capsule.Risk + ". " +
		"Allowed tools: " + strings.Join(capsule.AllowedTools, ", ") + ". " +
		"Forbidden tools: " + strings.Join(capsule.ForbiddenTools, ", ") + ". " +
		"Success checks: " + strings.Join(capsule.SuccessChecks, "; ") + "."
	resp, err := agent.NewService(s.harnessConfig(), s.db).Chat(ctx, agent.ChatRequest{
		SessionID: capsule.ID + "_planner",
		UserID:    agent.DefaultUserID,
		Channel:   "harness",
		Message:   prompt,
		Mode:      capsule.Mode,
	})
	if err != nil {
		return Plan{}, resp.Provider, resp.Model, err
	}
	plan := parsePlan(resp.Reply)
	plan.Goal = capsule.Goal
	plan.Mode = capsule.Mode
	plan.Risk = capsule.Risk
	if plan.Summary == "" {
		plan.Summary = "đã tạo plan harness"
	}
	if len(plan.Steps) == 0 {
		_ = s.addEvent(ctx, capsule.ID, "plan.parse_failed", mustJSON(map[string]any{
			"text_len": len(resp.Reply),
			"preview":  truncate(resp.Reply, 600),
		}))
		plan = fallbackPlan(capsule, "provider returned an empty plan")
	}
	return plan, resp.Provider, resp.Model, nil
}

func (s *Service) prepareWorktree(ctx context.Context, capsule *Capsule) error {
	if err := os.MkdirAll(filepath.Dir(capsule.WorktreePath), 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(capsule.WorktreePath); err == nil {
		return nil
	}
	result, err := runCommand(ctx, capsule.WorkspaceRoot, []string{"git", "worktree", "add", "-b", capsule.BranchName, capsule.WorktreePath, capsule.BaseRef})
	_ = s.addEvent(ctx, capsule.ID, "worktree.create", mustJSON(result))
	if err != nil {
		return fmt.Errorf("create worktree: %s", result.Output)
	}
	return nil
}

func (s *Service) contextPack(ctx context.Context, capsule Capsule) map[string]any {
	status, _ := runCommand(ctx, capsule.WorkspaceRoot, []string{"git", "status", "--short"})
	files := limitedFiles(capsule.WorktreePath, 160)
	candidates := candidateFiles(ctx, capsule.WorktreePath, capsule.Goal, files)
	docs := importantDocs(capsule.WorktreePath)
	return map[string]any{
		"workspace_root":  capsule.WorkspaceRoot,
		"worktree_path":   capsule.WorktreePath,
		"git_status":      truncate(status.Output, 4000),
		"files":           files,
		"candidate_files": candidates,
		"docs":            docs,
	}
}

func (s *Service) localize(ctx context.Context, capsule Capsule, contextPack map[string]any, attempt int) map[string]any {
	testCommand := inferTestCommand(capsule.WorktreePath)
	return map[string]any{
		"attempt":         attempt,
		"candidate_files": contextPack["candidate_files"],
		"test_command":    testCommand,
	}
}

func (s *Service) patchWithProvider(ctx context.Context, capsule Capsule, contextPack map[string]any, localized map[string]any, lastVerify commandResult, attempt int) (string, string, string, error) {
	fileContext := readCandidateContext(capsule.WorktreePath, localized["candidate_files"], 24000)
	prompt := "Bạn là VietClaw Harness patcher. Chỉ trả unified diff hợp lệ, không markdown. " +
		"Không dùng secret/network/delete/push/deploy. " +
		"Goal: " + capsule.Goal + "\n" +
		"Attempt: " + fmt.Sprint(attempt) + "\n" +
		"Files:\n" + fileContext + "\n" +
		"Last verify:\n" + truncate(lastVerify.Output, 4000) + "\n"
	resp, err := agent.NewService(s.harnessConfig(), s.db).Chat(ctx, agent.ChatRequest{
		SessionID: capsule.ID + "_patcher",
		UserID:    agent.DefaultUserID,
		Channel:   "harness",
		Message:   prompt,
		Mode:      capsule.Mode,
	})
	if err != nil {
		return "", resp.Provider, resp.Model, err
	}
	_ = s.addEvent(ctx, capsule.ID, "model.patch", mustJSON(map[string]any{"provider": resp.Provider, "model": resp.Model, "attempt": attempt, "preview": truncate(resp.Reply, 1200)}))
	diff := extractDiff(resp.Reply)
	if strings.TrimSpace(diff) == "" {
		return "", resp.Provider, resp.Model, fmt.Errorf("model returned no unified diff")
	}
	return diff, resp.Provider, resp.Model, nil
}

func (s *Service) applyPatch(ctx context.Context, capsule Capsule, patchText string, attempt int) ([]string, error) {
	if !strings.Contains(patchText, "\n--- ") && !strings.HasPrefix(patchText, "--- ") && !strings.Contains(patchText, "\ndiff --git ") && !strings.HasPrefix(patchText, "diff --git ") {
		return nil, fmt.Errorf("patch is not a unified diff")
	}
	if !strings.HasSuffix(patchText, "\n") {
		patchText += "\n"
	}
	cmd := exec.CommandContext(ctx, "git", "apply", "--whitespace=nowarn")
	cmd.Dir = capsule.WorktreePath
	cmd.Stdin = strings.NewReader(patchText)
	out, err := cmd.CombinedOutput()
	result := commandResult{Command: "git apply --whitespace=nowarn", CWD: capsule.WorktreePath, ExitCode: exitCode(err), Output: string(out)}
	_ = s.addEvent(ctx, capsule.ID, "command.git_apply", mustJSON(map[string]any{"attempt": attempt, "result": result}))
	if err != nil {
		return nil, fmt.Errorf("%s", result.Output)
	}
	return s.changedFiles(ctx, capsule), nil
}

func (s *Service) verify(ctx context.Context, capsule Capsule, localized map[string]any) commandResult {
	command := fmt.Sprint(localized["test_command"])
	if blocked := forbiddenText(command); blocked != "" {
		return commandResult{Command: command, CWD: capsule.WorktreePath, ExitCode: 126, Output: blocked}
	}
	if !allowedTestCommand(command) {
		return commandResult{Command: command, CWD: capsule.WorktreePath, ExitCode: 126, Output: "test command not allowed"}
	}
	args := shellFields(command)
	result, err := runCommand(ctx, capsule.WorktreePath, args)
	if err != nil && result.ExitCode == 0 {
		result.ExitCode = exitCode(err)
	}
	if result.ExitCode != 0 && hasGoMod(capsule.WorktreePath) && command != "go test ./..." {
		fallback, _ := runCommand(ctx, capsule.WorktreePath, []string{"go", "test", "./..."})
		fallback.Command = "go test ./..."
		return fallback
	}
	return result
}

func (s *Service) finalDiff(ctx context.Context, capsule Capsule) string {
	result, _ := runCommand(ctx, capsule.WorktreePath, []string{"git", "diff", "--no-ext-diff", "--"})
	return result.Output
}

func (s *Service) changedFiles(ctx context.Context, capsule Capsule) []string {
	result, _ := runCommand(ctx, capsule.WorktreePath, []string{"git", "diff", "--name-only", "--"})
	lines := strings.Split(strings.TrimSpace(result.Output), "\n")
	out := []string{}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	sort.Strings(out)
	return out
}

func (s *Service) load(ctx context.Context, id string) (Capsule, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT id, session_id, goal, mode, risk, status, budget_json, allowed_tools_json,
       forbidden_tools_json, success_checks_json, provider, model, summary, plan_json,
       workspace_root, worktree_path, branch_name, base_ref, changed_files_json, final_diff, failure_reason,
       created_at, updated_at
FROM harness_runs
WHERE id = ?`, id)
	return scanCapsule(row)
}

func (s *Service) block(ctx context.Context, capsule Capsule, reason string) (Capsule, error) {
	capsule.Status = StatusBlocked
	capsule.FailureReason = reason
	capsule.Summary = reason
	if err := s.updateExecution(ctx, capsule); err != nil {
		return Capsule{}, err
	}
	_ = s.updateStatus(ctx, capsule.ID, StatusBlocked, reason)
	_ = s.addEvent(ctx, capsule.ID, "run.blocked", mustJSON(map[string]any{"reason": reason}))
	return s.load(ctx, capsule.ID)
}

func (s *Service) fail(ctx context.Context, capsule Capsule, reason string) (Capsule, error) {
	capsule.Status = StatusFailed
	capsule.FailureReason = reason
	capsule.Summary = reason
	capsule.FinalDiff = s.finalDiff(ctx, capsule)
	capsule.ChangedFiles = s.changedFiles(ctx, capsule)
	if err := s.updateExecution(ctx, capsule); err != nil {
		return Capsule{}, err
	}
	_ = s.updateStatus(ctx, capsule.ID, StatusFailed, reason)
	_ = s.addEvent(ctx, capsule.ID, "run.failed", mustJSON(map[string]any{"reason": reason}))
	return s.load(ctx, capsule.ID)
}

func (s *Service) updateStatus(ctx context.Context, id string, status Status, failureReason string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `
UPDATE harness_runs
SET status = ?, failure_reason = CASE WHEN ? = '' THEN failure_reason ELSE ? END, updated_at = ?
WHERE id = ?`, status, failureReason, failureReason, now, id)
	return err
}

func (s *Service) updateExecution(ctx context.Context, capsule Capsule) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `
UPDATE harness_runs
SET status = ?, provider = ?, model = ?, summary = ?, plan_json = ?,
    workspace_root = ?, worktree_path = ?, branch_name = ?, base_ref = ?,
    changed_files_json = ?, final_diff = ?, failure_reason = ?, updated_at = ?
WHERE id = ?`,
		capsule.Status, nullable(capsule.Provider), nullable(capsule.Model), nullable(capsule.Summary),
		mustJSON(capsule.Plan), nullable(capsule.WorkspaceRoot), nullable(capsule.WorktreePath),
		nullable(capsule.BranchName), nullable(capsule.BaseRef), mustJSON(capsule.ChangedFiles),
		nullable(capsule.FinalDiff), nullable(capsule.FailureReason), now, capsule.ID)
	return err
}

func normalizeCreateRequest(req CreateRequest) CreateRequest {
	req.Goal = strings.TrimSpace(req.Goal)
	req.Mode = strings.ToLower(strings.TrimSpace(req.Mode))
	req.Risk = strings.ToLower(strings.TrimSpace(req.Risk))
	if req.Mode == "" {
		req.Mode = defaultMode
	}
	if req.Risk == "" {
		req.Risk = inferRisk(req)
	}
	if req.MaxTokens <= 0 {
		req.MaxTokens = defaultMaxTokens
	}
	if req.MaxUSD <= 0 {
		req.MaxUSD = defaultMaxUSD
	}
	if req.MaxMinutes <= 0 {
		req.MaxMinutes = defaultMaxMinutes
	}
	if len(req.AllowedTools) == 0 {
		req.AllowedTools = []string{"file.read", "file.patch", "test.run"}
	}
	if len(req.ForbiddenTools) == 0 {
		req.ForbiddenTools = []string{"secret.read", "shell.network", "shell.delete", "git.push"}
	}
	if len(req.SuccessChecks) == 0 {
		req.SuccessChecks = []string{"focused tests pass", "diff stays inside requested scope", "evidence ledger has command/test results"}
	}
	return req
}

func inferRisk(req CreateRequest) string {
	text := strings.ToLower(req.Goal + " " + strings.Join(req.AllowedTools, " "))
	highMarkers := []string{"push", "deploy", "delete", "secret", "token", "network", "shell", "production", "prod"}
	for _, marker := range highMarkers {
		if strings.Contains(text, marker) {
			return "high"
		}
	}
	if strings.Contains(text, "docs") || strings.Contains(text, "readme") || strings.Contains(text, "test") {
		return "low"
	}
	return defaultRisk
}

func fallbackPlan(capsule Capsule, reason string) Plan {
	return Plan{
		Goal:    capsule.Goal,
		Mode:    capsule.Mode,
		Risk:    capsule.Risk,
		Summary: reason,
		Assumptions: []string{
			"provider plan không khả dụng nên dùng plan an toàn mặc định",
		},
		Steps: []PlanStep{
			{ID: "step_1", Title: "localize", Description: "xác định file và context liên quan trước khi sửa", Tools: []string{"file.read"}},
			{ID: "step_2", Title: "patch", Description: "áp dụng thay đổi nhỏ theo đúng scope", Tools: []string{"file.patch"}},
			{ID: "step_3", Title: "verify", Description: "chạy check tập trung và ghi bằng chứng", Tools: []string{"test.run"}, Checks: capsule.SuccessChecks},
		},
		StopRules: []string{"dừng nếu cần secret/network/shell ngoài allowlist", "dừng nếu diff vượt scope"},
	}
}

func parsePlan(text string) Plan {
	cleaned := strings.TrimSpace(text)
	if start := strings.Index(cleaned, "{"); start >= 0 {
		cleaned = cleaned[start:]
	}
	if end := strings.LastIndex(cleaned, "}"); end >= 0 {
		cleaned = cleaned[:end+1]
	}
	var plan Plan
	if err := json.Unmarshal([]byte(cleaned), &plan); err == nil && len(plan.Steps) > 0 {
		normalizePlanSteps(&plan)
		return plan
	}
	var flexible struct {
		Summary     string         `json:"summary"`
		Assumptions []string       `json:"assumptions"`
		Steps       map[string]any `json:"steps"`
		StopRules   []string       `json:"stop_rules"`
	}
	if err := json.Unmarshal([]byte(cleaned), &flexible); err == nil {
		plan.Summary = flexible.Summary
		plan.Assumptions = flexible.Assumptions
		plan.StopRules = flexible.StopRules
		for _, key := range []string{"localize", "patch", "verify"} {
			if value, ok := flexible.Steps[key]; ok {
				plan.Steps = append(plan.Steps, PlanStep{ID: key, Title: key, Description: stepDescription(value)})
			}
		}
	}
	return plan
}

func normalizePlanSteps(plan *Plan) {
	for i := range plan.Steps {
		step := &plan.Steps[i]
		if step.ID == "" && step.Name != "" {
			step.ID = strings.ToLower(strings.TrimSpace(step.Name))
		}
		if step.Title == "" && step.Name != "" {
			step.Title = step.Name
		}
		if step.Description == "" && step.Detail != "" {
			step.Description = step.Detail
		}
		if step.Description == "" && step.Action != "" {
			step.Description = step.Action
		}
		if step.ID == "" {
			step.ID = fmt.Sprintf("step_%d", i+1)
		}
		if step.Title == "" {
			step.Title = step.ID
		}
		step.Name = ""
		step.Detail = ""
		step.Action = ""
	}
}

func stepDescription(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case map[string]any:
		for _, key := range []string{"description", "action", "summary"} {
			if text, ok := typed[key].(string); ok && text != "" {
				return text
			}
		}
		data, _ := json.Marshal(typed)
		return string(data)
	default:
		data, _ := json.Marshal(typed)
		return string(data)
	}
}

func truncate(value string, max int) string {
	if max <= 0 || len(value) <= max {
		return value
	}
	return value[:max]
}

type scanner interface {
	Scan(dest ...any) error
}

func scanCapsule(row scanner) (Capsule, error) {
	var capsule Capsule
	var sessionID, providerID, model, summary sql.NullString
	var workspaceRoot, worktreePath, branchName, baseRef, finalDiff, failureReason sql.NullString
	var budgetJSON, allowedJSON, forbiddenJSON, checksJSON, planJSON, changedFilesJSON string
	if err := row.Scan(&capsule.ID, &sessionID, &capsule.Goal, &capsule.Mode, &capsule.Risk,
		&capsule.Status, &budgetJSON, &allowedJSON, &forbiddenJSON, &checksJSON,
		&providerID, &model, &summary, &planJSON,
		&workspaceRoot, &worktreePath, &branchName, &baseRef, &changedFilesJSON, &finalDiff, &failureReason,
		&capsule.CreatedAt, &capsule.UpdatedAt); err != nil {
		return Capsule{}, err
	}
	capsule.SessionID = sessionID.String
	capsule.Provider = providerID.String
	capsule.Model = model.String
	capsule.Summary = summary.String
	capsule.WorkspaceRoot = workspaceRoot.String
	capsule.WorktreePath = worktreePath.String
	capsule.BranchName = branchName.String
	capsule.BaseRef = baseRef.String
	capsule.FinalDiff = finalDiff.String
	capsule.FailureReason = failureReason.String
	_ = json.Unmarshal([]byte(budgetJSON), &capsule.Budget)
	_ = json.Unmarshal([]byte(allowedJSON), &capsule.AllowedTools)
	_ = json.Unmarshal([]byte(forbiddenJSON), &capsule.ForbiddenTools)
	_ = json.Unmarshal([]byte(checksJSON), &capsule.SuccessChecks)
	_ = json.Unmarshal([]byte(planJSON), &capsule.Plan)
	_ = json.Unmarshal([]byte(changedFilesJSON), &capsule.ChangedFiles)
	return capsule, nil
}

func (s *Service) insertRun(ctx context.Context, capsule Capsule) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO harness_runs (
  id, session_id, goal, mode, risk, status, budget_json, allowed_tools_json,
  forbidden_tools_json, success_checks_json, provider, model, summary, plan_json,
  workspace_root, worktree_path, branch_name, base_ref, changed_files_json, final_diff, failure_reason,
  created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		capsule.ID, nullable(capsule.SessionID), capsule.Goal, capsule.Mode, capsule.Risk, capsule.Status,
		mustJSON(capsule.Budget), mustJSON(capsule.AllowedTools), mustJSON(capsule.ForbiddenTools),
		mustJSON(capsule.SuccessChecks), nullable(capsule.Provider), nullable(capsule.Model),
		nullable(capsule.Summary), "{}", nullable(capsule.WorkspaceRoot), nullable(capsule.WorktreePath),
		nullable(capsule.BranchName), nullable(capsule.BaseRef), mustJSON(capsule.ChangedFiles),
		nullable(capsule.FinalDiff), nullable(capsule.FailureReason), capsule.CreatedAt, capsule.UpdatedAt)
	return err
}

func (s *Service) updateRunPlan(ctx context.Context, capsule Capsule) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.ExecContext(ctx, `
UPDATE harness_runs
SET status = ?, provider = ?, model = ?, summary = ?, plan_json = ?, updated_at = ?
WHERE id = ?`,
		capsule.Status, nullable(capsule.Provider), nullable(capsule.Model), nullable(capsule.Summary),
		mustJSON(capsule.Plan), now, capsule.ID)
	return err
}

func (s *Service) addEvent(ctx context.Context, runID, eventType, payload string) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO harness_events (run_id, type, payload, created_at)
VALUES (?, ?, ?, ?)`, runID, eventType, payload, time.Now().UTC().Format(time.RFC3339))
	return err
}

func (s *Service) events(ctx context.Context, runID string) ([]Event, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT id, run_id, type, payload, created_at
FROM harness_events
WHERE run_id = ?
ORDER BY id ASC`, runID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []Event
	for rows.Next() {
		var event Event
		if err := rows.Scan(&event.ID, &event.RunID, &event.Type, &event.Payload, &event.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, event)
	}
	return out, rows.Err()
}

func mustJSON(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return "{}"
	}
	return string(data)
}

func nullable(value string) sql.NullString {
	return sql.NullString{String: value, Valid: strings.TrimSpace(value) != ""}
}

func newID(prefix string) string {
	var data [8]byte
	if _, err := rand.Read(data[:]); err != nil {
		return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
	}
	return prefix + "_" + hex.EncodeToString(data[:])
}

type commandResult struct {
	Command  string `json:"command"`
	CWD      string `json:"cwd"`
	ExitCode int    `json:"exit_code"`
	Output   string `json:"output"`
}

func (s *Service) dataDir() string {
	if s.cfg.Database.Path != "" {
		return filepath.Dir(s.cfg.Database.Path)
	}
	if paths, err := config.DefaultPaths(); err == nil {
		return paths.DataDir
	}
	return ".vietclaw"
}

func resolveWorkspaceRoot(value string, cfg config.Config) (string, error) {
	root := strings.TrimSpace(value)
	if root == "" {
		root = cfg.Agent.Workspace
	}
	if root == "" {
		return "", fmt.Errorf("workspace root is empty")
	}
	root = config.ExpandPath(root)
	abs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	if info, err := os.Stat(abs); err != nil || !info.IsDir() {
		return "", fmt.Errorf("workspace root not found: %s", abs)
	}
	return abs, nil
}

func gitTopLevel(ctx context.Context, root string) (string, error) {
	result, err := runCommand(ctx, root, []string{"git", "rev-parse", "--show-toplevel"})
	if err != nil || result.ExitCode != 0 {
		return "", fmt.Errorf("not a git repo")
	}
	return strings.TrimSpace(result.Output), nil
}

func runCommand(ctx context.Context, cwd string, args []string) (commandResult, error) {
	if len(args) == 0 {
		return commandResult{CWD: cwd, ExitCode: 127, Output: "empty command"}, fmt.Errorf("empty command")
	}
	commandText := strings.Join(args, " ")
	cmdCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(cmdCtx, args[0], args[1:]...)
	cmd.Dir = cwd
	out, err := cmd.CombinedOutput()
	result := commandResult{
		Command:  commandText,
		CWD:      cwd,
		ExitCode: exitCode(err),
		Output:   truncate(string(out), 16000),
	}
	if cmdCtx.Err() == context.DeadlineExceeded {
		result.ExitCode = 124
		result.Output += "\ncommand timeout"
		return result, cmdCtx.Err()
	}
	return result, err
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	if exit, ok := err.(*exec.ExitError); ok {
		return exit.ExitCode()
	}
	return 1
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func forbiddenText(text string) string {
	lower := strings.ToLower(text)
	for _, marker := range []string{
		"secret.read", "env_get", "git push", " push ", "deploy", "rm -rf", "remove-item", "del /s",
		"curl ", "wget ", "http://", "https://", "shell.network", "shell.delete",
	} {
		if strings.Contains(lower, marker) {
			return "blocked forbidden operation: " + marker
		}
	}
	return ""
}

func allowedTestCommand(command string) bool {
	fields := shellFields(command)
	if len(fields) == 0 {
		return false
	}
	switch fields[0] {
	case "go":
		return len(fields) >= 2 && fields[1] == "test"
	case "npm", "pnpm", "yarn":
		return len(fields) >= 2 && (fields[1] == "test" || fields[1] == "run")
	case "cargo":
		return len(fields) >= 2 && fields[1] == "test"
	case "pytest":
		return true
	default:
		return false
	}
}

func shellFields(command string) []string {
	return strings.Fields(strings.TrimSpace(command))
}

func inferTestCommand(root string) string {
	if hasGoMod(root) {
		return "go test ./..."
	}
	for _, name := range []string{"package.json", "pnpm-lock.yaml", "yarn.lock"} {
		if _, err := os.Stat(filepath.Join(root, name)); err == nil {
			if name == "pnpm-lock.yaml" {
				return "pnpm test"
			}
			if name == "yarn.lock" {
				return "yarn test"
			}
			return "npm test"
		}
	}
	if _, err := os.Stat(filepath.Join(root, "Cargo.toml")); err == nil {
		return "cargo test"
	}
	return "go test ./..."
}

func hasGoMod(root string) bool {
	_, err := os.Stat(filepath.Join(root, "go.mod"))
	return err == nil
}

func limitedFiles(root string, limit int) []string {
	out := []string{}
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || len(out) >= limit {
			return nil
		}
		name := d.Name()
		if d.IsDir() {
			if name == ".git" || name == "node_modules" || name == "target" || name == "dist" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err == nil {
			out = append(out, filepath.ToSlash(rel))
		}
		return nil
	})
	sort.Strings(out)
	return out
}

func candidateFiles(ctx context.Context, root, goal string, files []string) []string {
	terms := goalTerms(goal)
	selected := []string{}
	for _, file := range files {
		lower := strings.ToLower(file)
		for _, term := range terms {
			if strings.Contains(lower, term) {
				selected = append(selected, file)
				break
			}
		}
		if len(selected) >= 12 {
			return selected
		}
	}
	if len(selected) == 0 {
		for _, suffix := range []string{"_test.go", ".test.ts", ".test.js", ".go", ".ts", ".js"} {
			for _, file := range files {
				if strings.HasSuffix(file, suffix) {
					selected = append(selected, file)
					if len(selected) >= 12 {
						return selected
					}
				}
			}
		}
	}
	if len(selected) == 0 {
		selected = files
		if len(selected) > 12 {
			selected = selected[:12]
		}
	}
	_ = ctx
	_ = root
	return selected
}

func goalTerms(goal string) []string {
	raw := strings.FieldsFunc(strings.ToLower(goal), func(r rune) bool {
		return r < 'a' || r > 'z'
	})
	out := []string{}
	for _, term := range raw {
		if len(term) >= 3 && term != "fix" && term != "test" && term != "failing" {
			out = append(out, term)
		}
	}
	return out
}

func importantDocs(root string) []string {
	docs := []string{}
	for _, name := range []string{"AGENTS.md", "README.md", "go.mod", "package.json"} {
		path := filepath.Join(root, name)
		if data, err := os.ReadFile(path); err == nil {
			docs = append(docs, name+":\n"+truncate(string(data), 3000))
		}
	}
	return docs
}

func readCandidateContext(root string, value any, maxChars int) string {
	var files []string
	if list, ok := value.([]string); ok {
		files = list
	} else if list, ok := value.([]any); ok {
		for _, item := range list {
			if text, ok := item.(string); ok {
				files = append(files, text)
			}
		}
	}
	var b strings.Builder
	for _, file := range files {
		clean := filepath.Clean(file)
		if strings.HasPrefix(clean, "..") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, clean))
		if err != nil {
			continue
		}
		chunk := "\n--- " + filepath.ToSlash(clean) + " ---\n" + string(data)
		if b.Len()+len(chunk) > maxChars {
			break
		}
		b.WriteString(chunk)
	}
	return b.String()
}

func extractDiff(text string) string {
	cleaned := strings.TrimSpace(text)
	if strings.Contains(cleaned, "```") {
		parts := strings.Split(cleaned, "```")
		for _, part := range parts {
			part = strings.TrimSpace(strings.TrimPrefix(part, "diff"))
			if strings.HasPrefix(part, "diff --git ") || strings.HasPrefix(part, "--- ") {
				return strings.TrimSpace(part)
			}
		}
	}
	if idx := strings.Index(cleaned, "diff --git "); idx >= 0 {
		return cleaned[idx:]
	}
	if idx := strings.Index(cleaned, "--- "); idx >= 0 {
		return cleaned[idx:]
	}
	return cleaned
}

func (s *Service) cancelled(ctx context.Context, id string) bool {
	var status string
	_ = s.db.QueryRowContext(ctx, "SELECT status FROM harness_runs WHERE id = ?", id).Scan(&status)
	return status == string(StatusCancelled)
}
