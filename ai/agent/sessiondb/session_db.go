package sessiondb

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/lmorg/ttyphoon/app"
)

const (
	driverName          = "sqlite3"
	defaultSessionTitle = "New session"
	defaultHistoryLimit = 24
)

var mu sync.Mutex

type Entry struct {
	ID          int64
	Prompt      string
	CommandLine string
	OutputBlock string
	LLMResponse string
}

type FrontendHistoryItemT struct {
	ID          int64  `json:"id"`
	Prompt      string `json:"prompt"`
	CommandLine string `json:"commandLine"`
	OutputBlock string `json:"outputBlock"`
	Response    string `json:"response"`
	Excerpt     string `json:"excerpt"`
}

type FrontendSessionMetaT struct {
	TableID    int64  `json:"tableId"`
	Summary    string `json:"summary"`
	Created    string `json:"created"`
	Updated    string `json:"updated"`
	Active     bool   `json:"active"`
	EntryCount int    `json:"entryCount"`
}

type FrontendStateT struct {
	ActiveSessionID int64                  `json:"activeSessionId"`
	Sessions        []FrontendSessionMetaT `json:"sessions"`
	History         []FrontendHistoryItemT `json:"history"`
}

func GetFrontendState(workspace string, limit int) (FrontendStateT, error) {
	mu.Lock()
	defer mu.Unlock()

	if limit <= 0 {
		limit = defaultHistoryLimit
	}

	db, err := openDB(workspace)
	if err != nil {
		return FrontendStateT{}, err
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return FrontendStateT{}, fmt.Errorf("cannot begin sessiondb transaction: %w", err)
	}
	defer tx.Rollback()

	activeID, err := ensureActiveSessionTx(tx)
	if err != nil {
		return FrontendStateT{}, err
	}

	state, err := frontendStateTx(tx, activeID, limit)
	if err != nil {
		return FrontendStateT{}, err
	}

	if err := tx.Commit(); err != nil {
		return FrontendStateT{}, fmt.Errorf("cannot commit sessiondb transaction: %w", err)
	}

	return state, nil
}

func CreateSession(workspace string, summary string, limit int) (FrontendStateT, error) {
	mu.Lock()
	defer mu.Unlock()

	db, err := openDB(workspace)
	if err != nil {
		return FrontendStateT{}, err
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return FrontendStateT{}, fmt.Errorf("cannot begin sessiondb transaction: %w", err)
	}
	defer tx.Rollback()

	meta, err := createSessionTx(tx, sessionSummary(summary), true)
	if err != nil {
		return FrontendStateT{}, err
	}

	state, err := frontendStateTx(tx, meta.TableID, limit)
	if err != nil {
		return FrontendStateT{}, err
	}

	if err := tx.Commit(); err != nil {
		return FrontendStateT{}, fmt.Errorf("cannot commit sessiondb transaction: %w", err)
	}

	return state, nil
}

func SetActiveSession(workspace string, tableID int64, limit int) (FrontendStateT, error) {
	mu.Lock()
	defer mu.Unlock()

	db, err := openDB(workspace)
	if err != nil {
		return FrontendStateT{}, err
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return FrontendStateT{}, fmt.Errorf("cannot begin sessiondb transaction: %w", err)
	}
	defer tx.Rollback()

	if err := setActiveSessionTx(tx, tableID); err != nil {
		return FrontendStateT{}, err
	}

	state, err := frontendStateTx(tx, tableID, limit)
	if err != nil {
		return FrontendStateT{}, err
	}

	if err := tx.Commit(); err != nil {
		return FrontendStateT{}, fmt.Errorf("cannot commit sessiondb transaction: %w", err)
	}

	return state, nil
}

func DeleteSession(workspace string, tableID int64, limit int) (FrontendStateT, error) {
	mu.Lock()
	defer mu.Unlock()

	db, err := openDB(workspace)
	if err != nil {
		return FrontendStateT{}, err
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return FrontendStateT{}, fmt.Errorf("cannot begin sessiondb transaction: %w", err)
	}
	defer tx.Rollback()

	wasActive, err := sessionIsActiveTx(tx, tableID)
	if err != nil {
		return FrontendStateT{}, err
	}

	if _, err := tx.Exec(fmt.Sprintf(`DROP TABLE IF EXISTS %s`, quotedSessionTable(tableID))); err != nil {
		return FrontendStateT{}, fmt.Errorf("cannot drop session table %d: %w", tableID, err)
	}

	if _, err := tx.Exec(`DELETE FROM sessions_meta WHERE tableId = ?`, tableID); err != nil {
		return FrontendStateT{}, fmt.Errorf("cannot delete session metadata %d: %w", tableID, err)
	}

	activeID, err := activeSessionIDTx(tx)
	if err != nil {
		return FrontendStateT{}, err
	}
	if wasActive || activeID == 0 {
		activeID, err = activateLatestOrCreateTx(tx)
		if err != nil {
			return FrontendStateT{}, err
		}
	}

	state, err := frontendStateTx(tx, activeID, limit)
	if err != nil {
		return FrontendStateT{}, err
	}

	if err := tx.Commit(); err != nil {
		return FrontendStateT{}, fmt.Errorf("cannot commit sessiondb transaction: %w", err)
	}

	return state, nil
}

func RenameSession(workspace string, tableID int64, summary string, limit int) (FrontendStateT, error) {
	mu.Lock()
	defer mu.Unlock()

	db, err := openDB(workspace)
	if err != nil {
		return FrontendStateT{}, err
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return FrontendStateT{}, fmt.Errorf("cannot begin sessiondb transaction: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`UPDATE sessions_meta SET summary = ?, updated = ? WHERE tableId = ?`, sessionSummary(summary), nowString(), tableID); err != nil {
		return FrontendStateT{}, fmt.Errorf("cannot rename session %d: %w", tableID, err)
	}

	activeID, err := activeSessionIDTx(tx)
	if err != nil {
		return FrontendStateT{}, err
	}

	state, err := frontendStateTx(tx, activeID, limit)
	if err != nil {
		return FrontendStateT{}, err
	}

	if err := tx.Commit(); err != nil {
		return FrontendStateT{}, fmt.Errorf("cannot commit sessiondb transaction: %w", err)
	}

	return state, nil
}

func ClearActiveSession(workspace string, limit int) (FrontendStateT, error) {
	mu.Lock()
	defer mu.Unlock()

	db, err := openDB(workspace)
	if err != nil {
		return FrontendStateT{}, err
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return FrontendStateT{}, fmt.Errorf("cannot begin sessiondb transaction: %w", err)
	}
	defer tx.Rollback()

	activeID, err := ensureActiveSessionTx(tx)
	if err != nil {
		return FrontendStateT{}, err
	}

	if _, err := tx.Exec(fmt.Sprintf(`DELETE FROM %s`, quotedSessionTable(activeID))); err != nil {
		return FrontendStateT{}, fmt.Errorf("cannot clear active session history: %w", err)
	}

	if _, err := tx.Exec(`UPDATE sessions_meta SET updated = ?, entryCount = 0 WHERE tableId = ?`, nowString(), activeID); err != nil {
		return FrontendStateT{}, fmt.Errorf("cannot update active session timestamp: %w", err)
	}

	state, err := frontendStateTx(tx, activeID, limit)
	if err != nil {
		return FrontendStateT{}, err
	}

	if err := tx.Commit(); err != nil {
		return FrontendStateT{}, fmt.Errorf("cannot commit sessiondb transaction: %w", err)
	}

	return state, nil
}

// DeleteActiveSessionEntry removes a single prompt/response row (by its id, the
// same value used as the per-prompt log file's promptID) from the active
// session's history table.
func DeleteActiveSessionEntry(workspace string, entryID int64, limit int) (FrontendStateT, error) {
	mu.Lock()
	defer mu.Unlock()

	db, err := openDB(workspace)
	if err != nil {
		return FrontendStateT{}, err
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return FrontendStateT{}, fmt.Errorf("cannot begin sessiondb transaction: %w", err)
	}
	defer tx.Rollback()

	activeID, err := ensureActiveSessionTx(tx)
	if err != nil {
		return FrontendStateT{}, err
	}

	res, err := tx.Exec(fmt.Sprintf(`DELETE FROM %s WHERE id = ?`, quotedSessionTable(activeID)), entryID)
	if err != nil {
		return FrontendStateT{}, fmt.Errorf("cannot delete session entry %d: %w", entryID, err)
	}

	if affected, _ := res.RowsAffected(); affected > 0 {
		if _, err := tx.Exec(`UPDATE sessions_meta SET updated = ?, entryCount = MAX(entryCount - 1, 0) WHERE tableId = ?`, nowString(), activeID); err != nil {
			return FrontendStateT{}, fmt.Errorf("cannot update active session metadata: %w", err)
		}
	}

	state, err := frontendStateTx(tx, activeID, limit)
	if err != nil {
		return FrontendStateT{}, err
	}

	if err := tx.Commit(); err != nil {
		return FrontendStateT{}, fmt.Errorf("cannot commit sessiondb transaction: %w", err)
	}

	return state, nil
}

// AppendActiveSessionEntry inserts a new prompt/response row into the active session's history
// table and returns the promptID (the row's autoincrement id) alongside the active session id.
func AppendActiveSessionEntry(workspace, prompt, commandLine, outputBlock, llmResponse string) (sessionID, promptID int64, err error) {
	mu.Lock()
	defer mu.Unlock()

	db, err := openDB(workspace)
	if err != nil {
		return 0, 0, err
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return 0, 0, fmt.Errorf("cannot begin sessiondb transaction: %w", err)
	}
	defer tx.Rollback()

	activeID, err := ensureActiveSessionTx(tx)
	if err != nil {
		return 0, 0, err
	}

	res, err := tx.Exec(
		fmt.Sprintf(`INSERT INTO %s (prompt, command_line, output_block, llm_response) VALUES (?, ?, ?, ?)`, quotedSessionTable(activeID)),
		strings.TrimSpace(prompt),
		strings.TrimSpace(commandLine),
		strings.TrimSpace(outputBlock),
		strings.TrimSpace(llmResponse),
	)
	if err != nil {
		return 0, 0, fmt.Errorf("cannot insert session history row: %w", err)
	}

	insertedID, err := res.LastInsertId()
	if err != nil {
		return 0, 0, fmt.Errorf("cannot resolve inserted prompt id: %w", err)
	}

	if _, err := tx.Exec(
		`UPDATE sessions_meta SET updated = ?, entryCount = entryCount + 1, summary = CASE WHEN TRIM(COALESCE(summary, '')) = '' OR summary = ? THEN ? ELSE summary END WHERE tableId = ?`,
		nowString(),
		defaultSessionTitle,
		sessionSummary(prompt),
		activeID,
	); err != nil {
		return 0, 0, fmt.Errorf("cannot update active session metadata: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, 0, fmt.Errorf("cannot commit sessiondb transaction: %w", err)
	}

	return activeID, insertedID, nil
}

func ActiveSessionEntries(workspace string, limit int) ([]Entry, error) {
	mu.Lock()
	defer mu.Unlock()

	db, err := openDB(workspace)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("cannot begin sessiondb transaction: %w", err)
	}
	defer tx.Rollback()

	activeID, err := ensureActiveSessionTx(tx)
	if err != nil {
		return nil, err
	}

	entries, err := loadEntriesTx(tx, activeID, limit)
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("cannot commit sessiondb transaction: %w", err)
	}

	return entries, nil
}

func ActiveSessionID(workspace string) (int64, error) {
	mu.Lock()
	defer mu.Unlock()

	db, err := openDB(workspace)
	if err != nil {
		return 0, err
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return 0, fmt.Errorf("cannot begin sessiondb transaction: %w", err)
	}
	defer tx.Rollback()

	activeID, err := ensureActiveSessionTx(tx)
	if err != nil {
		return 0, err
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("cannot commit sessiondb transaction: %w", err)
	}

	return activeID, nil
}

func frontendStateTx(tx *sql.Tx, activeID int64, limit int) (FrontendStateT, error) {
	sessions, err := loadSessionMetasTx(tx)
	if err != nil {
		return FrontendStateT{}, err
	}

	history, err := loadFrontendHistoryTx(tx, activeID, limit)
	if err != nil {
		return FrontendStateT{}, err
	}

	return FrontendStateT{
		ActiveSessionID: activeID,
		Sessions:        sessions,
		History:         history,
	}, nil
}

func loadSessionMetasTx(tx *sql.Tx) ([]FrontendSessionMetaT, error) {
	// entryCount is maintained incrementally (see AppendActiveSessionEntry and
	// ClearActiveSession) rather than recomputed here, so opening the session
	// list never scans a history table.
	rows, err := tx.Query(`
		SELECT tableId, summary, created, updated, active, entryCount
		FROM sessions_meta
		ORDER BY updated DESC, tableId DESC`)
	if err != nil {
		return nil, fmt.Errorf("cannot query session metadata: %w", err)
	}
	defer rows.Close()

	out := make([]FrontendSessionMetaT, 0)
	for rows.Next() {
		var meta FrontendSessionMetaT
		var activeInt int
		if err := rows.Scan(&meta.TableID, &meta.Summary, &meta.Created, &meta.Updated, &activeInt, &meta.EntryCount); err != nil {
			return nil, fmt.Errorf("cannot scan session metadata: %w", err)
		}

		meta.Active = activeInt == 1
		out = append(out, meta)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("cannot iterate session metadata: %w", err)
	}

	return out, nil
}

// maxHistoryFieldChars caps fields rendered directly into the AI Settings
// history list. A field this large is virtually always a stray base64 blob
// (e.g. an image data URL that was fed through the generic "ask AI about this
// document" path rather than the dedicated image attachment path); rendering
// it as one giant unbroken string of text is what froze the modal, since
// browsers are pathologically slow laying out whitespace-free text of that
// length. This does not affect what the model saw or what is stored on disk.
const maxHistoryFieldChars = 4000

func truncateHistoryField(s string) string {
	if len(s) <= maxHistoryFieldChars {
		return s
	}
	return s[:maxHistoryFieldChars] + "\n\n[truncated for display]"
}

func loadFrontendHistoryTx(tx *sql.Tx, tableID int64, limit int) ([]FrontendHistoryItemT, error) {
	entries, err := loadEntriesTx(tx, tableID, limit)
	if err != nil {
		return nil, err
	}

	// loadEntriesTx returns chronological order for context reconstruction
	// (ActiveSessionEntries); the frontend transcript wants newest first.
	out := make([]FrontendHistoryItemT, 0, len(entries))
	for i := len(entries) - 1; i >= 0; i-- {
		entry := entries[i]
		response := strings.TrimSpace(entry.LLMResponse)
		excerpt := strings.Join(strings.Fields(response), " ")
		if len(excerpt) > 180 {
			excerpt = strings.TrimSpace(excerpt[:180])
		}

		out = append(out, FrontendHistoryItemT{
			ID:          entry.ID,
			Prompt:      truncateHistoryField(strings.TrimSpace(entry.Prompt)),
			CommandLine: truncateHistoryField(strings.TrimSpace(entry.CommandLine)),
			OutputBlock: truncateHistoryField(strings.TrimSpace(entry.OutputBlock)),
			Response:    response,
			Excerpt:     excerpt,
		})
	}

	return out, nil
}

func loadEntriesTx(tx *sql.Tx, tableID int64, limit int) ([]Entry, error) {
	if tableID <= 0 {
		return []Entry{}, nil
	}

	if limit <= 0 {
		limit = defaultHistoryLimit
	}

	rows, err := tx.Query(
		fmt.Sprintf(`SELECT id, prompt, command_line, output_block, llm_response FROM %s ORDER BY id DESC LIMIT ?`, quotedSessionTable(tableID)),
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("cannot query session entries: %w", err)
	}
	defer rows.Close()

	out := make([]Entry, 0)
	for rows.Next() {
		var entry Entry
		if err := rows.Scan(&entry.ID, &entry.Prompt, &entry.CommandLine, &entry.OutputBlock, &entry.LLMResponse); err != nil {
			return nil, fmt.Errorf("cannot scan session entry: %w", err)
		}
		out = append(out, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("cannot iterate session entries: %w", err)
	}

	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}

	return out, nil
}

func openDB(workspace string) (*sql.DB, error) {
	path, err := dbPath(workspace)
	if err != nil {
		return nil, err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("cannot create session database directory: %w", err)
	}

	db, err := sql.Open(driverName, path)
	if err != nil {
		return nil, fmt.Errorf("cannot open session database: %w", err)
	}

	if err := initDB(db); err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func initDB(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS sessions_meta (
			tableId INTEGER PRIMARY KEY AUTOINCREMENT,
			summary TEXT NOT NULL,
			created TEXT NOT NULL,
			updated TEXT NOT NULL,
			active INTEGER NOT NULL DEFAULT 0 CHECK (active IN (0, 1)),
			entryCount INTEGER NOT NULL DEFAULT 0
		);
		CREATE UNIQUE INDEX IF NOT EXISTS sessions_meta_active_idx ON sessions_meta(active) WHERE active = 1;
	`)
	if err != nil {
		return fmt.Errorf("cannot initialize session metadata table: %w", err)
	}

	return migrateEntryCountColumn(db)
}

// migrateEntryCountColumn adds entryCount to a sessions_meta table created before
// the column existed, backfilling it once from the per-session history tables.
// Reads afterwards use this maintained column instead of scanning every
// session's history table on every call, which is what previously made opening
// AI Settings slow to the point of appearing hung for workspaces with a large
// history.
func migrateEntryCountColumn(db *sql.DB) error {
	rows, err := db.Query(`PRAGMA table_info(sessions_meta)`)
	if err != nil {
		return fmt.Errorf("cannot inspect sessions_meta schema: %w", err)
	}

	hasColumn := false
	for rows.Next() {
		var (
			cid        int
			name       string
			ctype      string
			notNull    int
			defaultVal any
			pk         int
		)
		if err := rows.Scan(&cid, &name, &ctype, &notNull, &defaultVal, &pk); err != nil {
			rows.Close()
			return fmt.Errorf("cannot read sessions_meta column info: %w", err)
		}
		if name == "entryCount" {
			hasColumn = true
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("cannot iterate sessions_meta column info: %w", err)
	}
	rows.Close()

	if hasColumn {
		return nil
	}

	if _, err := db.Exec(`ALTER TABLE sessions_meta ADD COLUMN entryCount INTEGER NOT NULL DEFAULT 0`); err != nil {
		return fmt.Errorf("cannot add entryCount column: %w", err)
	}

	tableIDs, err := db.Query(`SELECT tableId FROM sessions_meta`)
	if err != nil {
		return fmt.Errorf("cannot list sessions to backfill entryCount: %w", err)
	}
	defer tableIDs.Close()

	ids := make([]int64, 0)
	for tableIDs.Next() {
		var id int64
		if err := tableIDs.Scan(&id); err != nil {
			return fmt.Errorf("cannot scan session id to backfill entryCount: %w", err)
		}
		ids = append(ids, id)
	}
	if err := tableIDs.Err(); err != nil {
		return fmt.Errorf("cannot iterate sessions to backfill entryCount: %w", err)
	}

	// One-time backfill: every existing session is scanned exactly once here,
	// after which entryCount is kept in sync incrementally.
	for _, id := range ids {
		var count int
		if err := db.QueryRow(fmt.Sprintf(`SELECT COUNT(1) FROM %s`, quotedSessionTable(id))).Scan(&count); err != nil {
			return fmt.Errorf("cannot backfill entryCount for session %d: %w", id, err)
		}
		if _, err := db.Exec(`UPDATE sessions_meta SET entryCount = ? WHERE tableId = ?`, count, id); err != nil {
			return fmt.Errorf("cannot store entryCount for session %d: %w", id, err)
		}
	}

	return nil
}

func ensureActiveSessionTx(tx *sql.Tx) (int64, error) {
	activeID, err := activeSessionIDTx(tx)
	if err != nil {
		return 0, err
	}
	if activeID != 0 {
		return activeID, nil
	}

	return activateLatestOrCreateTx(tx)
}

func activeSessionIDTx(tx *sql.Tx) (int64, error) {
	var tableID int64
	err := tx.QueryRow(`SELECT tableId FROM sessions_meta WHERE active = 1 LIMIT 1`).Scan(&tableID)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("cannot query active session: %w", err)
	}
	return tableID, nil
}

func activateLatestOrCreateTx(tx *sql.Tx) (int64, error) {
	var tableID int64
	err := tx.QueryRow(`SELECT tableId FROM sessions_meta ORDER BY updated DESC, tableId DESC LIMIT 1`).Scan(&tableID)
	if err == sql.ErrNoRows {
		meta, createErr := createSessionTx(tx, defaultSessionTitle, true)
		if createErr != nil {
			return 0, createErr
		}
		return meta.TableID, nil
	}
	if err != nil {
		return 0, fmt.Errorf("cannot query fallback session: %w", err)
	}

	if err := setActiveSessionTx(tx, tableID); err != nil {
		return 0, err
	}

	return tableID, nil
}

func createSessionTx(tx *sql.Tx, summary string, active bool) (FrontendSessionMetaT, error) {
	if active {
		if _, err := tx.Exec(`UPDATE sessions_meta SET active = 0 WHERE active = 1`); err != nil {
			return FrontendSessionMetaT{}, fmt.Errorf("cannot clear active session flag: %w", err)
		}
	}

	now := nowString()
	res, err := tx.Exec(
		`INSERT INTO sessions_meta (summary, created, updated, active) VALUES (?, ?, ?, ?)`,
		sessionSummary(summary),
		now,
		now,
		boolToInt(active),
	)
	if err != nil {
		return FrontendSessionMetaT{}, fmt.Errorf("cannot create session metadata: %w", err)
	}

	tableID, err := res.LastInsertId()
	if err != nil {
		return FrontendSessionMetaT{}, fmt.Errorf("cannot read session metadata id: %w", err)
	}

	if _, err := tx.Exec(fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			prompt TEXT NOT NULL,
			command_line TEXT NOT NULL,
			output_block TEXT NOT NULL,
			llm_response TEXT NOT NULL
		)`, quotedSessionTable(tableID))); err != nil {
		return FrontendSessionMetaT{}, fmt.Errorf("cannot create session table %d: %w", tableID, err)
	}

	return FrontendSessionMetaT{TableID: tableID, Summary: sessionSummary(summary), Created: now, Updated: now, Active: active}, nil
}

func setActiveSessionTx(tx *sql.Tx, tableID int64) error {
	exists, err := sessionExistsTx(tx, tableID)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("session %d does not exist", tableID)
	}

	if _, err := tx.Exec(`UPDATE sessions_meta SET active = 0 WHERE active = 1`); err != nil {
		return fmt.Errorf("cannot clear active session flag: %w", err)
	}

	// Selecting a session is not an update to it: `updated` reflects prompt
	// activity only, so the list order isn't disturbed by merely switching to it.
	if _, err := tx.Exec(`UPDATE sessions_meta SET active = 1 WHERE tableId = ?`, tableID); err != nil {
		return fmt.Errorf("cannot activate session %d: %w", tableID, err)
	}

	return nil
}

func sessionExistsTx(tx *sql.Tx, tableID int64) (bool, error) {
	var exists int
	err := tx.QueryRow(`SELECT 1 FROM sessions_meta WHERE tableId = ? LIMIT 1`, tableID).Scan(&exists)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("cannot query session %d: %w", tableID, err)
	}
	return true, nil
}

func sessionIsActiveTx(tx *sql.Tx, tableID int64) (bool, error) {
	var activeInt int
	err := tx.QueryRow(`SELECT active FROM sessions_meta WHERE tableId = ?`, tableID).Scan(&activeInt)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("cannot query session activity: %w", err)
	}
	return activeInt == 1, nil
}

func dbPath(workspace string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot resolve home directory: %w", err)
	}

	name := sanitizeWorkspace(workspace)
	return filepath.Join(homeDir, "Documents", app.DirName, fmt.Sprintf("session.%s.db", name)), nil
}

func sanitizeWorkspace(workspace string) string {
	name := strings.TrimSpace(workspace)
	if name == "" {
		return "default"
	}

	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		" ", "_",
		"\t", "_",
	)
	name = replacer.Replace(name)
	name = strings.Trim(name, "._")
	if name == "" {
		return "default"
	}
	return name
}

func quotedSessionTable(tableID int64) string {
	return fmt.Sprintf(`"session_%d"`, tableID)
}

func sessionSummary(summary string) string {
	summary = strings.Join(strings.Fields(strings.TrimSpace(summary)), " ")
	if summary == "" {
		return defaultSessionTitle
	}
	if len(summary) > 80 {
		return strings.TrimSpace(summary[:80])
	}
	return summary
}

func nowString() string {
	return time.Now().UTC().Format(time.RFC3339Nano)
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
