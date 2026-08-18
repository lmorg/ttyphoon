package sessiondb

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/lmorg/ttyphoon/app"
)

const (
	SESSION_LOG_START_JOB = iota + 1
	SESSION_LOG_APPEND_CHUNK
	SESSION_LOG_FINALIZE_JOB
	SESSION_LOG_FINISH_JOB
)

// SessionLogContext is passed to WriteToSessionLog for every state transition.
// PromptID is only meaningful on SESSION_LOG_FINALIZE_JOB: it's the id assigned
// by the sessions DB after the LLM call returned, and is used to rename the
// pending markdown file to its final per-prompt name.
type SessionLogContext struct {
	Workspace       string
	Query           string
	CommandLine     string
	OutputBlock     string
	PromptID        int64
	WorkspaceActive bool
	Emit            func(event string, payload any)
}

// sessionLogState tracks the currently-streaming prompt for a workspace so
// chunks are written to disk in order and the pending file can be renamed on
// finalize.
type sessionLogState struct {
	workspace   string
	sessionID   int64
	pendingPath string
	streamed    strings.Builder
	requestOpen bool
}

var aiSessionLogStore = struct {
	sync.Mutex
	byWorkspace map[string]*sessionLogState
}{
	byWorkspace: map[string]*sessionLogState{},
}

func normalizeWorkspaceName(workspace string) string {
	w := strings.TrimSpace(workspace)
	if w == "" {
		return "default"
	}
	return w
}

func sessionLogDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(homeDir, "Documents", app.DirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func sessionLogPendingPath(workspace string, sessionID int64) (string, error) {
	if sessionID <= 0 {
		return "", fmt.Errorf("invalid session id %d", sessionID)
	}
	dir, err := sessionLogDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, fmt.Sprintf("session-log.%s.%d.pending.md", sanitizeWorkspace(workspace), sessionID)), nil
}

func sessionLogPromptPath(workspace string, sessionID, promptID int64) (string, error) {
	if sessionID <= 0 || promptID <= 0 {
		return "", fmt.Errorf("invalid session %d / prompt %d", sessionID, promptID)
	}
	dir, err := sessionLogDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, fmt.Sprintf("session-log.%s.%d.%d.md", sanitizeWorkspace(workspace), sessionID, promptID)), nil
}

// sessionLogPromptFilesGlob matches every per-prompt log file for a session
// (including the pending file, which callers filter out when needed).
func sessionLogPromptFilesGlob(workspace string, sessionID int64) (string, error) {
	if sessionID <= 0 {
		return "", fmt.Errorf("invalid session id %d", sessionID)
	}
	dir, err := sessionLogDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, fmt.Sprintf("session-log.%s.%d.*.md", sanitizeWorkspace(workspace), sessionID)), nil
}

func formatSessionLogTimestamp(ts time.Time) string {
	return ts.Format("02-01-06 15:04")
}

func summarizeRequestHeading(query string) string {
	trimmed := strings.Join(strings.Fields(strings.TrimSpace(query)), " ")
	if trimmed == "" {
		return "AI request"
	}
	if len(trimmed) > 96 {
		return strings.TrimSpace(trimmed[:96]) + "..."
	}
	return trimmed
}

func appendSessionLog(path, text string) error {
	if text == "" {
		return nil
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = f.WriteString(text)
	return err
}

func writePromptLogHeader(path, workspace string, sessionID int64, now time.Time) error {
	header := fmt.Sprintf(
		"# AI Session Log\n\nWorkspace: %s\n\nSession ID: %d\n\nStarted: %s\n",
		normalizeWorkspaceName(workspace),
		sessionID,
		formatSessionLogTimestamp(now),
	)
	return os.WriteFile(path, []byte(header), 0o644)
}

func buildSessionLogRequestPrefix(query, commandLine, outputBlock string, now time.Time) string {
	var b strings.Builder
	b.WriteString("\n\n<!-- start request: ")
	b.WriteString(formatSessionLogTimestamp(now))
	b.WriteString(" -->\n")
	b.WriteString("<!-- request heading: ")
	b.WriteString(summarizeRequestHeading(query))
	b.WriteString(" -->\n")
	b.WriteString("## Request\n\n")

	trimmedQuery := strings.TrimSpace(query)
	if trimmedQuery != "" {
		b.WriteString("### Prompt\n\n")
		b.WriteString(trimmedQuery)
		b.WriteString("\n\n")
	}

	trimmedCommand := strings.TrimSpace(commandLine)
	if trimmedCommand != "" {
		b.WriteString("### Command\n\n~~~sh\n")
		b.WriteString(trimmedCommand)
		b.WriteString("\n~~~\n\n")
	}

	trimmedOutput := strings.TrimSpace(head(outputBlock))
	if trimmedOutput != "" {
		b.WriteString("### Output\n\n~~~text\n")
		b.WriteString(trimmedOutput)
		b.WriteString("\n~~~\n\n")
	}

	b.WriteString("### Stream Trace\n\n<!-- stream trace: start -->\n")
	return b.String()
}

const maxLines = 20

func head(s string) string {
	slice := strings.SplitN(s, "\n", maxLines+1)
	if len(slice) > maxLines {
		slice = slice[:maxLines]
	}
	return strings.Join(slice, "\n")
}

func buildSessionLogFinalizeSuffix(streamed, finalOutput string, now time.Time) string {
	var b strings.Builder
	b.WriteString("\n<!-- stream trace: end -->\n")

	// When nothing was streamed we add a Response section for the final output.
	// When content was streamed, the full response already lives inside the
	// stream trace above and repeating it here would duplicate the answer.
	if strings.TrimSpace(streamed) == "" {
		if trimmedFinal := strings.TrimSpace(finalOutput); trimmedFinal != "" {
			b.WriteString("\n### Response\n\n")
			b.WriteString(trimmedFinal)
			b.WriteString("\n")
		}
	}

	b.WriteString("\n<!-- end request: ")
	b.WriteString(formatSessionLogTimestamp(now))
	b.WriteString(" -->\n")
	return b.String()
}

// ensureRequestOpenLocked writes the request header/prefix exactly once per
// request. It is a no-op when the request is already open, which prevents the
// prompt block from being written twice (start job + first chunk).
// Returns the prefix that was written, or an empty string when it was a no-op.
func ensureRequestOpenLocked(state *sessionLogState, ctx SessionLogContext, now time.Time) string {
	if state == nil || state.requestOpen {
		return ""
	}
	// Fresh pending file — overwrite anything left behind from an interrupted run.
	if err := writePromptLogHeader(state.pendingPath, state.workspace, state.sessionID, now); err != nil {
		return ""
	}
	state.streamed.Reset()
	state.requestOpen = true
	prefix := buildSessionLogRequestPrefix(ctx.Query, ctx.CommandLine, ctx.OutputBlock, now)
	_ = appendSessionLog(state.pendingPath, prefix)
	return prefix
}

func activeSessionLogStateLocked(workspace string) (*sessionLogState, error) {
	ws := normalizeWorkspaceName(workspace)

	sessionID, err := ActiveSessionID(ws)
	if err != nil {
		return nil, err
	}

	state := aiSessionLogStore.byWorkspace[ws]
	if state != nil && state.sessionID == sessionID {
		return state, nil
	}

	pendingPath, err := sessionLogPendingPath(ws, sessionID)
	if err != nil {
		return nil, err
	}

	state = &sessionLogState{
		workspace:   ws,
		sessionID:   sessionID,
		pendingPath: pendingPath,
	}
	aiSessionLogStore.byWorkspace[ws] = state
	return state, nil
}

// GetSessionLog returns the markdown content of the most recent finalized
// prompt log for the active session in the given workspace. Used by the AI
// panel to render the "current" prompt when a workspace is opened.
func GetSessionLog(workspace string) string {
	ws := normalizeWorkspaceName(workspace)
	sessionID, err := ActiveSessionID(ws)
	if err != nil || sessionID <= 0 {
		return ""
	}
	metas := listPromptLogMetas(ws, sessionID)
	if len(metas) == 0 {
		return ""
	}
	return GetPromptLog(ws, sessionID, metas[len(metas)-1].PromptID)
}

// GetPromptLog returns the markdown for a specific prompt within a session.
// Returns an empty string when the file is missing.
func GetPromptLog(workspace string, sessionID, promptID int64) string {
	path, err := sessionLogPromptPath(workspace, sessionID, promptID)
	if err != nil {
		return ""
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return string(b)
}

// PromptLogMeta describes a single per-prompt log file. Emitted to the
// frontend so the AI panel can populate its Prompt dropdown.
type PromptLogMeta struct {
	SessionID int64  `json:"sessionId"`
	PromptID  int64  `json:"promptId"`
	Heading   string `json:"heading"`
	SizeBytes int64  `json:"sizeBytes"`
}

var promptLogFilenameRx = regexp.MustCompile(`^session-log\..+\.(\d+)\.(\d+)\.md$`)

func listPromptLogMetas(workspace string, sessionID int64) []PromptLogMeta {
	glob, err := sessionLogPromptFilesGlob(workspace, sessionID)
	if err != nil {
		return nil
	}
	entries, err := filepath.Glob(glob)
	if err != nil {
		return nil
	}

	pendingName := ""
	if pendingPath, pErr := sessionLogPendingPath(workspace, sessionID); pErr == nil {
		pendingName = filepath.Base(pendingPath)
	}

	metas := make([]PromptLogMeta, 0, len(entries))
	for _, path := range entries {
		base := filepath.Base(path)
		if base == pendingName {
			continue
		}
		m := promptLogFilenameRx.FindStringSubmatch(base)
		if len(m) != 3 {
			continue
		}
		var sid, pid int64
		if _, err := fmt.Sscanf(m[1], "%d", &sid); err != nil {
			continue
		}
		if _, err := fmt.Sscanf(m[2], "%d", &pid); err != nil {
			continue
		}
		if sid != sessionID {
			continue
		}
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		metas = append(metas, PromptLogMeta{
			SessionID: sid,
			PromptID:  pid,
			Heading:   readRequestHeadingComment(path),
			SizeBytes: info.Size(),
		})
	}

	// Sort ascending by prompt id (chronological).
	for i := 1; i < len(metas); i++ {
		for j := i; j > 0 && metas[j-1].PromptID > metas[j].PromptID; j-- {
			metas[j-1], metas[j] = metas[j], metas[j-1]
		}
	}
	return metas
}

var requestHeadingCommentRx = regexp.MustCompile(`(?m)^<!-- request heading: (.*) -->`)

// readRequestHeadingComment extracts the "<!-- request heading: ... -->" tag
// written by buildSessionLogRequestPrefix so the dropdown can show the prompt
// summary without a DB round-trip. Falls back to an empty string on failure.
func readRequestHeadingComment(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	// Prefix lives near the top of the file, no need to read past ~4KB.
	buf := make([]byte, 4096)
	n, _ := f.Read(buf)
	m := requestHeadingCommentRx.FindStringSubmatch(string(buf[:n]))
	if len(m) != 2 {
		return ""
	}
	return strings.TrimSpace(m[1])
}

// ListPromptLogs returns per-prompt log metadata for the active session in the
// given workspace, ordered chronologically.
func ListPromptLogs(workspace string) []PromptLogMeta {
	ws := normalizeWorkspaceName(workspace)
	sessionID, err := ActiveSessionID(ws)
	if err != nil || sessionID <= 0 {
		return nil
	}
	return listPromptLogMetas(ws, sessionID)
}

// ClearActiveSessionLog removes every per-prompt log file (plus any pending
// file) for the currently active session in the workspace.
func ClearActiveSessionLog(workspace string) error {
	ws := normalizeWorkspaceName(workspace)
	sessionID, err := ActiveSessionID(ws)
	if err != nil {
		return err
	}
	return ClearSessionLog(ws, sessionID)
}

// ClearSessionLog removes every per-prompt log file (plus any pending file) for
// the given session in the workspace.
func ClearSessionLog(workspace string, sessionID int64) error {
	aiSessionLogStore.Lock()
	defer aiSessionLogStore.Unlock()

	ws := normalizeWorkspaceName(workspace)
	if state := aiSessionLogStore.byWorkspace[ws]; state != nil && state.sessionID == sessionID {
		state.streamed.Reset()
		delete(aiSessionLogStore.byWorkspace, ws)
	}

	glob, err := sessionLogPromptFilesGlob(workspace, sessionID)
	if err != nil {
		return err
	}
	matches, err := filepath.Glob(glob)
	if err != nil {
		return err
	}
	pendingPath, _ := sessionLogPendingPath(workspace, sessionID)
	if pendingPath != "" {
		matches = append(matches, pendingPath)
	}

	for _, path := range matches {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

// DeleteSessionLog is used when the session itself is deleted.
func DeleteSessionLog(workspace string, sessionID int64) error {
	return ClearSessionLog(workspace, sessionID)
}

// finalizePromptLogLocked appends the closing suffix to the pending file and
// renames it to its per-prompt destination. When promptID is 0 (unknown, e.g.
// a race), the pending file is left in place for the next run to overwrite.
func finalizePromptLogLocked(state *sessionLogState, suffix string, promptID int64) {
	if state == nil || state.pendingPath == "" {
		return
	}
	_ = appendSessionLog(state.pendingPath, suffix)
	state.streamed.Reset()
	state.requestOpen = false

	if promptID <= 0 {
		return
	}
	finalPath, err := sessionLogPromptPath(state.workspace, state.sessionID, promptID)
	if err != nil {
		return
	}
	// Rename atomically overwrites any earlier file with the same promptID.
	_ = os.Rename(state.pendingPath, finalPath)
}

func WriteToSessionLog(ctx SessionLogContext, state int, payload string) {
	workspace := normalizeWorkspaceName(ctx.Workspace)

	switch state {
	case SESSION_LOG_START_JOB:
		now := time.Now()
		var prefix string
		aiSessionLogStore.Lock()
		if ref, err := activeSessionLogStateLocked(workspace); err == nil && ref != nil {
			// Discard any pending file left over from an interrupted job.
			if ref.requestOpen {
				ref.streamed.Reset()
				ref.requestOpen = false
				_ = os.Remove(ref.pendingPath)
			}
			prefix = ensureRequestOpenLocked(ref, ctx, now)
		}
		aiSessionLogStore.Unlock()

		if ctx.WorkspaceActive && ctx.Emit != nil {
			ctx.Emit("aiJobStart", prefix)
		}

	case SESSION_LOG_APPEND_CHUNK:
		if payload == "" {
			return
		}

		now := time.Now()
		aiSessionLogStore.Lock()
		if ref, err := activeSessionLogStateLocked(workspace); err == nil && ref != nil {
			ensureRequestOpenLocked(ref, ctx, now)
			ref.streamed.WriteString(payload)
			_ = appendSessionLog(ref.pendingPath, payload)
		}
		aiSessionLogStore.Unlock()

		if ctx.WorkspaceActive && ctx.Emit != nil {
			ctx.Emit("aiResponseStream", payload)
		}

	case SESSION_LOG_FINALIZE_JOB:
		now := time.Now()
		var suffix string
		streamedEmpty := true
		aiSessionLogStore.Lock()
		if ref, err := activeSessionLogStateLocked(workspace); err == nil && ref != nil {
			ensureRequestOpenLocked(ref, ctx, now)
			streamedEmpty = strings.TrimSpace(ref.streamed.String()) == ""
			suffix = buildSessionLogFinalizeSuffix(ref.streamed.String(), payload, now)
			finalizePromptLogLocked(ref, suffix, ctx.PromptID)
		}
		aiSessionLogStore.Unlock()

		if streamedEmpty && suffix != "" && ctx.WorkspaceActive && ctx.Emit != nil {
			ctx.Emit("aiResponseStream", suffix)
		}

	case SESSION_LOG_FINISH_JOB:
		if ctx.WorkspaceActive && ctx.Emit != nil {
			ctx.Emit("aiJobFinish", nil)
		}
	}
}
