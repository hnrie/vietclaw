package db

import (
	"database/sql"
	"fmt"
)

func ApplySchema(database *sql.DB) error {
	if _, err := database.Exec(schema); err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}

	// Dynamic migration to add embedding column to memories if it does not exist
	_, _ = database.Exec(`ALTER TABLE memories ADD COLUMN embedding BLOB`)
	_, _ = database.Exec(`ALTER TABLE harness_runs ADD COLUMN workspace_root TEXT`)
	_, _ = database.Exec(`ALTER TABLE harness_runs ADD COLUMN worktree_path TEXT`)
	_, _ = database.Exec(`ALTER TABLE harness_runs ADD COLUMN branch_name TEXT`)
	_, _ = database.Exec(`ALTER TABLE harness_runs ADD COLUMN base_ref TEXT`)
	_, _ = database.Exec(`ALTER TABLE harness_runs ADD COLUMN changed_files_json TEXT NOT NULL DEFAULT '[]'`)
	_, _ = database.Exec(`ALTER TABLE harness_runs ADD COLUMN final_diff TEXT`)
	_, _ = database.Exec(`ALTER TABLE harness_runs ADD COLUMN failure_reason TEXT`)

	return nil
}
