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

	return nil
}
