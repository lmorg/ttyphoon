package sessiondb

import (
	"database/sql"
	"fmt"
)

func deleteStreamSessionTx(tx *sql.Tx, sessionID int64) error {
	if _, err := tx.Exec(`DELETE FROM stream_blocks WHERE sessionId = ?`, sessionID); err != nil {
		return fmt.Errorf("cannot delete stream blocks for session %d: %w", sessionID, err)
	}
	if _, err := tx.Exec(`DELETE FROM stream_prompts WHERE sessionId = ?`, sessionID); err != nil {
		return fmt.Errorf("cannot delete stream prompts for session %d: %w", sessionID, err)
	}
	return nil
}

func deleteStreamPromptTx(tx *sql.Tx, sessionID, promptID int64) error {
	if _, err := tx.Exec(`DELETE FROM stream_blocks WHERE sessionId = ? AND promptId = ?`, sessionID, promptID); err != nil {
		return fmt.Errorf("cannot delete stream blocks for prompt %d: %w", promptID, err)
	}
	if _, err := tx.Exec(`DELETE FROM stream_prompts WHERE sessionId = ? AND promptId = ?`, sessionID, promptID); err != nil {
		return fmt.Errorf("cannot delete stream prompt %d: %w", promptID, err)
	}
	return nil
}

func deleteStreamSession(workspace string, sessionID int64) error {
	return withStreamDB(workspace, func(db *sql.DB) error {
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("cannot begin stream session deletion: %w", err)
		}
		defer tx.Rollback()
		if err := deleteStreamSessionTx(tx, sessionID); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("cannot commit stream session deletion: %w", err)
		}
		return nil
	})
}

func deleteStreamPrompt(workspace string, sessionID, promptID int64) error {
	return withStreamDB(workspace, func(db *sql.DB) error {
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("cannot begin stream prompt deletion: %w", err)
		}
		defer tx.Rollback()
		if err := deleteStreamPromptTx(tx, sessionID, promptID); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("cannot commit stream prompt deletion: %w", err)
		}
		return nil
	})
}
