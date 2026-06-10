ALTER TABLE harness_runs ADD COLUMN workspace_root TEXT;
ALTER TABLE harness_runs ADD COLUMN worktree_path TEXT;
ALTER TABLE harness_runs ADD COLUMN branch_name TEXT;
ALTER TABLE harness_runs ADD COLUMN base_ref TEXT;
ALTER TABLE harness_runs ADD COLUMN changed_files_json TEXT NOT NULL DEFAULT '[]';
ALTER TABLE harness_runs ADD COLUMN final_diff TEXT;
ALTER TABLE harness_runs ADD COLUMN failure_reason TEXT;
