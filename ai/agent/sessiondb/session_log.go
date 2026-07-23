package sessiondb

import (
	"fmt"
	"os"
	"path/filepath"
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

type SessionLogContext struct {
	Workspace       string
	Query           string
	CommandLine     string
	OutputBlock     string
	WorkspaceActive bool
	Emit            func(event string, payload any)
}

type sessionLogState struct {
	workspace   string
	sessionID   int64
	path        string
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

func sessionLogPath(workspace string, sessionID int64) (string, error) {
	if sessionID <= 0 {
		return "", fmt.Errorf("invalid session id %d", sessionID)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	dir := filepath.Join(homeDir, "Documents", app.DirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	return filepath.Join(dir, fmt.Sprintf("session-log.%s.%d.md", sanitizeWorkspace(workspace), sessionID)), nil
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

func writeSessionLogHeaderIfNeeded(path, workspace string, sessionID int64) error {
	info, err := os.Stat(path)
	if err == nil && info.Size() > 0 {
		return nil
	}
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	header := fmt.Sprintf("# AI Session Log\n\nWorkspace: %s\n\nSession ID: %d\n", normalizeWorkspaceName(workspace), sessionID)
	return os.WriteFile(path, []byte(header), 0o644)
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

	// Only add a dedicated Response section when nothing was streamed (eg a
	// non-streaming completion). When content was streamed, the full response
	// already lives inside the stream trace above, so repeating it here would
	// duplicate the answer.
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

func finishActiveSessionLogLocked(state *sessionLogState, finalOutput string, now time.Time) {
	if state == nil || state.path == "" {
		return
	}

	_ = appendSessionLog(state.path, buildSessionLogFinalizeSuffix(state.streamed.String(), finalOutput, now))
	state.streamed.Reset()
	state.requestOpen = false
}

// ensureRequestOpenLocked writes the request header/prefix exactly once per
// request. It is a no-op when the request is already open, which prevents the
// prompt block from being written twice (start job + first chunk).
func ensureRequestOpenLocked(state *sessionLogState, ctx SessionLogContext, now time.Time) {
	if state == nil || state.requestOpen {
		return
	}
	if err := writeSessionLogHeaderIfNeeded(state.path, state.workspace, state.sessionID); err != nil {
		return
	}
	state.streamed.Reset()
	state.requestOpen = true
	_ = appendSessionLog(state.path, buildSessionLogRequestPrefix(ctx.Query, ctx.CommandLine, ctx.OutputBlock, now))
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

	path, err := sessionLogPath(ws, sessionID)
	if err != nil {
		return nil, err
	}

	state = &sessionLogState{
		workspace: ws,
		sessionID: sessionID,
		path:      path,
	}
	aiSessionLogStore.byWorkspace[ws] = state
	return state, nil
}

func GetSessionLog(workspace string) string {
	ws := normalizeWorkspaceName(workspace)
	sessionID, err := ActiveSessionID(ws)
	if err != nil || sessionID <= 0 {
		return ""
	}

	path, err := sessionLogPath(ws, sessionID)
	if err != nil {
		return ""
	}

	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	return string(b)
}

func ClearActiveSessionLog(workspace string) error {
	ws := normalizeWorkspaceName(workspace)
	sessionID, err := ActiveSessionID(ws)
	if err != nil {
		return err
	}
	return ClearSessionLog(ws, sessionID)
}

func ClearSessionLog(workspace string, sessionID int64) error {
	path, err := sessionLogPath(workspace, sessionID)
	if err != nil {
		return err
	}

	aiSessionLogStore.Lock()
	defer aiSessionLogStore.Unlock()

	ws := normalizeWorkspaceName(workspace)
	if state := aiSessionLogStore.byWorkspace[ws]; state != nil && state.sessionID == sessionID {
		state.streamed.Reset()
		delete(aiSessionLogStore.byWorkspace, ws)
	}

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

func DeleteSessionLog(workspace string, sessionID int64) error {
	return ClearSessionLog(workspace, sessionID)
}

func WriteToSessionLog(ctx SessionLogContext, state int, payload string) {
	workspace := normalizeWorkspaceName(ctx.Workspace)

	switch state {
	case SESSION_LOG_START_JOB:
		now := time.Now()
		aiSessionLogStore.Lock()
		if ref, err := activeSessionLogStateLocked(workspace); err == nil && ref != nil {
			// Close any request left open by an interrupted job before starting
			// a new one.
			if ref.requestOpen {
				finishActiveSessionLogLocked(ref, "", now)
			}
			ensureRequestOpenLocked(ref, ctx, now)
		}
		aiSessionLogStore.Unlock()

		// The panel renders the prompt from the job title, so the request prefix
		// is deliberately NOT streamed to the frontend (that would duplicate the
		// prompt on screen). It is only written to the log file.
		if ctx.WorkspaceActive && ctx.Emit != nil {
			ctx.Emit("aiJobStart", payload)
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
			_ = appendSessionLog(ref.path, payload)
		}
		aiSessionLogStore.Unlock()

		if ctx.WorkspaceActive && ctx.Emit != nil {
			ctx.Emit("aiResponseStream", payload)
		}

	case SESSION_LOG_FINALIZE_JOB:
		now := time.Now()
		streamedEmpty := true
		aiSessionLogStore.Lock()
		if ref, err := activeSessionLogStateLocked(workspace); err == nil && ref != nil {
			ensureRequestOpenLocked(ref, ctx, now)
			streamedEmpty = strings.TrimSpace(ref.streamed.String()) == ""
			finishActiveSessionLogLocked(ref, payload, now)
		}
		aiSessionLogStore.Unlock()

		// When nothing was streamed (eg a non-streaming completion) the panel
		// would otherwise be left blank, so surface the final answer once. When
		// content WAS streamed it is already on screen, so we emit nothing here
		// to avoid duplicating the response.
		if streamedEmpty && payload != "" && ctx.WorkspaceActive && ctx.Emit != nil {
			ctx.Emit("aiResponseStream", payload)
		}

	case SESSION_LOG_FINISH_JOB:
		if ctx.WorkspaceActive && ctx.Emit != nil {
			ctx.Emit("aiJobFinish", nil)
		}
	}
}
