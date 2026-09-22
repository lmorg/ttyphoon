package sessiondb

import (
	"database/sql"
	"fmt"
	"strings"
)

type StreamBlockKind string

const (
	StreamBlockRequest    StreamBlockKind = "request"
	StreamBlockText       StreamBlockKind = "text"
	StreamBlockThinking   StreamBlockKind = "thinking"
	StreamBlockToolCall   StreamBlockKind = "tool-call"
	StreamBlockToolOutput StreamBlockKind = "tool-output"
	StreamBlockToolError  StreamBlockKind = "tool-error"
	StreamBlockSummary    StreamBlockKind = "tool-summary"
	StreamBlockSubagent   StreamBlockKind = "subagent"
	StreamBlockNotice     StreamBlockKind = "notice"
	StreamBlockQuestion   StreamBlockKind = "question"
)

type AIStreamBlock struct {
	ID        int64           `json:"id"`
	SessionID int64           `json:"sessionId"`
	PromptID  int64           `json:"promptId"`
	RunID     uint64          `json:"runId"`
	Workspace string          `json:"workspace"`
	BlockID   string          `json:"blockId"`
	ParentID  string          `json:"parentId"`
	Kind      StreamBlockKind `json:"kind"`
	Label     string          `json:"label"`
	Ordinal   int64           `json:"ordinal"`
	Status    string          `json:"status"`
	Delta     string          `json:"delta"`
	Content   string          `json:"content"`
}

type StreamBlockMeta struct {
	SessionID int64           `json:"sessionId"`
	PromptID  int64           `json:"promptId"`
	RunID     uint64          `json:"runId"`
	BlockID   string          `json:"blockId"`
	ParentID  string          `json:"parentId"`
	Kind      StreamBlockKind `json:"kind"`
	Label     string          `json:"label"`
	Ordinal   int64           `json:"ordinal"`
	Status    string          `json:"status"`
	Size      int64           `json:"size"`
}

func ListStreamPromptMetas(workspace string, sessionID int64) ([]PromptLogMeta, error) {
	if sessionID <= 0 {
		return nil, fmt.Errorf("stream prompt session id must be positive")
	}

	var prompts []PromptLogMeta
	err := withStreamDB(workspace, func(db *sql.DB) error {
		rows, err := db.Query(`
			SELECT p.sessionId, p.promptId, p.heading, COALESCE(SUM(length(b.content)), 0)
			FROM stream_prompts p
			LEFT JOIN stream_blocks b
				ON b.sessionId = p.sessionId AND b.promptId = p.promptId
			WHERE p.sessionId = ? AND p.promptId > 0
			GROUP BY p.sessionId, p.promptId, p.heading
			ORDER BY p.promptId
		`, sessionID)
		if err != nil {
			return fmt.Errorf("cannot list stream prompts: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var prompt PromptLogMeta
			if err := rows.Scan(&prompt.SessionID, &prompt.PromptID, &prompt.Heading, &prompt.SizeBytes); err != nil {
				return fmt.Errorf("cannot scan stream prompt metadata: %w", err)
			}
			prompts = append(prompts, prompt)
		}
		return rows.Err()
	})
	return prompts, err
}

func GetStreamPromptContent(workspace string, sessionID, promptID int64) (string, error) {
	if sessionID <= 0 || promptID <= 0 {
		return "", fmt.Errorf("stream prompt session and prompt ids must be positive")
	}

	var content strings.Builder
	err := withStreamDB(workspace, func(db *sql.DB) error {
		rows, err := db.Query(`
			SELECT content
			FROM stream_blocks
			WHERE sessionId = ? AND promptId = ?
			ORDER BY ordinal, id
		`, sessionID, promptID)
		if err != nil {
			return fmt.Errorf("cannot load stream prompt blocks: %w", err)
		}
		defer rows.Close()

		found := false
		for rows.Next() {
			var blockContent string
			if err := rows.Scan(&blockContent); err != nil {
				return fmt.Errorf("cannot scan stream prompt block: %w", err)
			}
			found = true
			content.WriteString(blockContent)
		}
		if err := rows.Err(); err != nil {
			return err
		}
		if !found {
			return fmt.Errorf("stream prompt %d was not found", promptID)
		}
		return nil
	})
	return content.String(), err
}

func ListStreamBlockMeta(workspace string, sessionID, promptID int64) ([]StreamBlockMeta, error) {
	if sessionID <= 0 || promptID <= 0 {
		return nil, fmt.Errorf("stream block session and prompt ids must be positive")
	}

	var blocks []StreamBlockMeta
	err := withStreamDB(workspace, func(db *sql.DB) error {
		rows, err := db.Query(`
			SELECT sessionId, promptId, runId, blockId, parentId, kind, label, ordinal, status, length(content)
			FROM stream_blocks
			WHERE sessionId = ? AND promptId = ?
			ORDER BY ordinal, id
		`, sessionID, promptID)
		if err != nil {
			return fmt.Errorf("cannot list stream blocks: %w", err)
		}
		defer rows.Close()
		blocks, err = scanStreamBlockMeta(rows)
		return err
	})
	return blocks, err
}

// ListLiveStreamBlockMeta returns the blocks the panel should render as "live":
// the in-flight run when one is open, otherwise the newest finalized prompt.
func ListLiveStreamBlockMeta(workspace string) ([]StreamBlockMeta, error) {
	ws := normalizeWorkspaceName(workspace)
	sessionID, err := ActiveSessionID(ws)
	if err != nil {
		return nil, err
	}
	if sessionID <= 0 {
		return nil, nil
	}

	if identity := GetActiveStreamIdentity(ws); identity.SessionID == sessionID && identity.RunID > 0 {
		return listStreamBlockMetaByRun(ws, sessionID, identity.RunID)
	}

	metas := listPromptLogMetasMerged(ws, sessionID)
	if len(metas) == 0 {
		return nil, nil
	}
	return ListStreamBlockMeta(ws, sessionID, metas[len(metas)-1].PromptID)
}

func listStreamBlockMetaByRun(workspace string, sessionID int64, runID uint64) ([]StreamBlockMeta, error) {
	var blocks []StreamBlockMeta
	err := withStreamDB(workspace, func(db *sql.DB) error {
		rows, err := db.Query(`
			SELECT sessionId, promptId, runId, blockId, parentId, kind, label, ordinal, status, length(content)
			FROM stream_blocks
			WHERE sessionId = ? AND runId = ?
			ORDER BY ordinal, id
		`, sessionID, runID)
		if err != nil {
			return fmt.Errorf("cannot list live stream blocks: %w", err)
		}
		defer rows.Close()
		blocks, err = scanStreamBlockMeta(rows)
		return err
	})
	return blocks, err
}

func scanStreamBlockMeta(rows *sql.Rows) ([]StreamBlockMeta, error) {
	var blocks []StreamBlockMeta
	for rows.Next() {
		var block StreamBlockMeta
		var runID uint64
		if err := rows.Scan(&block.SessionID, &block.PromptID, &runID, &block.BlockID,
			&block.ParentID, &block.Kind, &block.Label, &block.Ordinal, &block.Status, &block.Size); err != nil {
			return nil, fmt.Errorf("cannot scan stream block metadata: %w", err)
		}
		block.RunID = runID
		blocks = append(blocks, block)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("cannot iterate stream block metadata: %w", err)
	}
	return blocks, nil
}

func GetStreamBlockContent(workspace string, sessionID, promptID int64, blockID string) (string, error) {
	// promptID is 0 while a run is in flight, which the live view reads before finalize.
	if sessionID <= 0 || promptID < 0 || strings.TrimSpace(blockID) == "" {
		return "", fmt.Errorf("stream block session, prompt, and block ids are required")
	}

	var content string
	err := withStreamDB(workspace, func(db *sql.DB) error {
		err := db.QueryRow(`
			SELECT content
			FROM stream_blocks
			WHERE sessionId = ? AND promptId = ? AND blockId = ?
		`, sessionID, promptID, blockID).Scan(&content)
		if err == sql.ErrNoRows {
			return fmt.Errorf("stream block %q was not found", blockID)
		}
		if err != nil {
			return fmt.Errorf("cannot load stream block %q: %w", blockID, err)
		}
		return nil
	})
	return content, err
}

func CreateStreamPrompt(workspace string, sessionID int64, promptID int64, runID uint64, heading, started string) error {
	return withStreamDB(workspace, func(db *sql.DB) error {
		_, err := db.Exec(`
			INSERT INTO stream_prompts (sessionId, promptId, runId, heading, started)
			VALUES (?, ?, ?, ?, ?)
		`, sessionID, promptID, runID, heading, started)
		if err != nil {
			return fmt.Errorf("cannot create stream prompt: %w", err)
		}
		return nil
	})
}

func CreateStreamBlock(workspace string, block AIStreamBlock, created string) error {
	if strings.TrimSpace(block.BlockID) == "" {
		return fmt.Errorf("stream block id is required")
	}
	if strings.TrimSpace(string(block.Kind)) == "" {
		return fmt.Errorf("stream block kind is required")
	}
	if block.SessionID <= 0 {
		return fmt.Errorf("stream block session id must be positive")
	}

	return withStreamDB(workspace, func(db *sql.DB) error {
		_, err := db.Exec(`
			INSERT INTO stream_blocks
				(sessionId, promptId, runId, blockId, parentId, kind, label, ordinal, status, content, created, updated)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		`, block.SessionID, block.PromptID, block.RunID, block.BlockID, block.ParentID,
			string(block.Kind), block.Label, block.Ordinal, streamBlockStatus(block.Status), block.Content, created, created)
		if err != nil {
			return fmt.Errorf("cannot create stream block %q: %w", block.BlockID, err)
		}
		return nil
	})
}

func AppendStreamBlock(workspace string, sessionID int64, runID uint64, blockID, delta, status, updated string) error {
	if sessionID <= 0 {
		return fmt.Errorf("stream block session id must be positive")
	}
	if strings.TrimSpace(blockID) == "" {
		return fmt.Errorf("stream block id is required")
	}
	if delta == "" && status == "" {
		return nil
	}

	return withStreamDB(workspace, func(db *sql.DB) error {
		result, err := db.Exec(`
			UPDATE stream_blocks
			SET content = content || ?,
				status = CASE WHEN ? = '' THEN status ELSE ? END,
				updated = ?
			WHERE sessionId = ? AND runId = ? AND blockId = ?
		`, delta, status, streamBlockStatus(status), updated, sessionID, runID, blockID)
		if err != nil {
			return fmt.Errorf("cannot append stream block %q: %w", blockID, err)
		}
		if count, countErr := result.RowsAffected(); countErr != nil {
			return fmt.Errorf("cannot inspect stream block %q update: %w", blockID, countErr)
		} else if count != 1 {
			return fmt.Errorf("stream block %q was not found", blockID)
		}
		return nil
	})
}

func FinalizeStreamPrompt(workspace string, sessionID int64, runID uint64, promptID int64, finished string) error {
	if sessionID <= 0 || promptID <= 0 {
		return fmt.Errorf("stream prompt session and prompt ids must be positive")
	}

	return withStreamDB(workspace, func(db *sql.DB) error {
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("cannot begin stream prompt finalization: %w", err)
		}
		defer tx.Rollback()

		if _, err := tx.Exec(`
			UPDATE stream_blocks
			SET promptId = ?
			WHERE sessionId = ? AND runId = ? AND promptId = 0
		`, promptID, sessionID, runID); err != nil {
			return fmt.Errorf("cannot finalize stream blocks: %w", err)
		}
		if _, err := tx.Exec(`
			UPDATE stream_prompts
			SET promptId = ?, finished = ?, blockCount = (
				SELECT COUNT(*) FROM stream_blocks WHERE sessionId = ? AND runId = ?
			)
			WHERE sessionId = ? AND promptId = 0 AND runId = ?
		`, promptID, finished, sessionID, runID, sessionID, runID); err != nil {
			return fmt.Errorf("cannot finalize stream prompt: %w", err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("cannot commit stream prompt finalization: %w", err)
		}
		return nil
	})
}

func withStreamDB(workspace string, fn func(*sql.DB) error) error {
	mu.Lock()
	defer mu.Unlock()

	db, err := openDB(workspace)
	if err != nil {
		return err
	}
	defer db.Close()
	return fn(db)
}

func streamBlockStatus(status string) string {
	if status == "closed" {
		return "closed"
	}
	return "open"
}
