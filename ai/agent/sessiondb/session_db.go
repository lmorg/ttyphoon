package sessiondb

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
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

	if _, err := tx.Exec(`UPDATE sessions_meta SET updated = ? WHERE tableId = ?`, nowString(), activeID); err != nil {
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

func AppendActiveSessionEntry(workspace, prompt, commandLine, outputBlock, llmResponse string) error {
	mu.Lock()
	defer mu.Unlock()

	db, err := openDB(workspace)
	if err != nil {
		return err
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("cannot begin sessiondb transaction: %w", err)
	}
	defer tx.Rollback()

	activeID, err := ensureActiveSessionTx(tx)
	if err != nil {
		return err
	}

	_, err = tx.Exec(
		fmt.Sprintf(`INSERT INTO %s (prompt, command_line, output_block, llm_response) VALUES (?, ?, ?, ?)`, quotedSessionTable(activeID)),
		strings.TrimSpace(prompt),
		strings.TrimSpace(commandLine),
		strings.TrimSpace(outputBlock),
		strings.TrimSpace(llmResponse),
	)
	if err != nil {
		return fmt.Errorf("cannot insert session history row: %w", err)
	}

	if _, err := tx.Exec(
		`UPDATE sessions_meta SET updated = ?, summary = CASE WHEN TRIM(COALESCE(summary, '')) = '' OR summary = ? THEN ? ELSE summary END WHERE tableId = ?`,
		nowString(),
		defaultSessionTitle,
		sessionSummary(prompt),
		activeID,
	); err != nil {
		return fmt.Errorf("cannot update active session metadata: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("cannot commit sessiondb transaction: %w", err)
	}

	return nil
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
	rows, err := tx.Query(`
		SELECT tableId, summary, created, updated, active
		FROM sessions_meta
		ORDER BY active DESC, updated DESC, tableId DESC`)
	if err != nil {
		return nil, fmt.Errorf("cannot query session metadata: %w", err)
	}
	defer rows.Close()

	out := make([]FrontendSessionMetaT, 0)
	for rows.Next() {
		var meta FrontendSessionMetaT
		var activeInt int
		if err := rows.Scan(&meta.TableID, &meta.Summary, &meta.Created, &meta.Updated, &activeInt); err != nil {
			return nil, fmt.Errorf("cannot scan session metadata: %w", err)
		}

		count, err := sessionEntryCountTx(tx, meta.TableID)
		if err != nil {
			return nil, err
		}

		meta.Active = activeInt == 1
		meta.EntryCount = count
		out = append(out, meta)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("cannot iterate session metadata: %w", err)
	}

	return out, nil
}

func loadFrontendHistoryTx(tx *sql.Tx, tableID int64, limit int) ([]FrontendHistoryItemT, error) {
	entries, err := loadEntriesTx(tx, tableID, limit)
	if err != nil {
		return nil, err
	}

	out := make([]FrontendHistoryItemT, 0, len(entries))
	for _, entry := range entries {
		response := strings.TrimSpace(entry.LLMResponse)
		excerpt := strings.Join(strings.Fields(response), " ")
		if len(excerpt) > 180 {
			excerpt = strings.TrimSpace(excerpt[:180])
		}

		out = append(out, FrontendHistoryItemT{
			ID:          entry.ID,
			Prompt:      strings.TrimSpace(entry.Prompt),
			CommandLine: strings.TrimSpace(entry.CommandLine),
			OutputBlock: strings.TrimSpace(entry.OutputBlock),
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
			active INTEGER NOT NULL DEFAULT 0 CHECK (active IN (0, 1))
		);
		CREATE UNIQUE INDEX IF NOT EXISTS sessions_meta_active_idx ON sessions_meta(active) WHERE active = 1;
	`)
	if err != nil {
		return fmt.Errorf("cannot initialize session metadata table: %w", err)
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

	if _, err := tx.Exec(`UPDATE sessions_meta SET active = 1, updated = ? WHERE tableId = ?`, nowString(), tableID); err != nil {
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

func sessionEntryCountTx(tx *sql.Tx, tableID int64) (int, error) {
	var count int
	err := tx.QueryRow(fmt.Sprintf(`SELECT COUNT(1) FROM %s`, quotedSessionTable(tableID))).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("cannot count session entries for %d: %w", tableID, err)
	}
	return count, nil
}

func dbPath(workspace string) (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot resolve home directory: %w", err)
	}

	name := sanitizeWorkspace(workspace)
	return filepath.Join(homeDir, "Documents", "ttyphoon", fmt.Sprintf("session.%s.db", name)), nil
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
