package sessiondb

import (
	"log"
	"strings"
	"sync"
	"time"
)

const (
	SESSION_LOG_START_JOB = iota + 1
	SESSION_LOG_FINALIZE_JOB
	SESSION_LOG_FINISH_JOB
)

// SessionLogContext is passed to WriteToSessionLog for every state transition.
// PromptID is only meaningful on SESSION_LOG_FINALIZE_JOB: it is the id assigned
// by the sessions DB after the LLM call returns and used to finalize SQLite
// stream rows.
type SessionLogContext struct {
	Workspace       string
	Query           string
	CommandLine     string
	OutputBlock     string
	PromptID        int64
	Status          string
	WorkspaceActive bool
	Emit            func(event string, payload any)
}

type AIJobStart struct {
	RunID uint64 `json:"runId"`
	Title string `json:"title"`
}

type AIJobFinish struct {
	RunID    uint64 `json:"runId"`
	PromptID int64  `json:"promptId"`
	Status   string `json:"status"`
}

// sessionLogState tracks the current prompt's lifecycle; durable content is
// stored in SQLite stream blocks.
type sessionLogState struct {
	workspace string
	sessionID int64
	active    bool
	runID     uint64
	promptID  int64
	status    string
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

// panelView tracks whether the AI panel is following live output, keyed by
// workspace: concurrent agents run in different workspaces, so a user reading
// history in one must not suppress a live run in another.
var panelView = struct {
	sync.Mutex
	byWorkspace map[string]bool
}{byWorkspace: map[string]bool{}}

// SetPanelView is called by the frontend when the AI panel switches between live
// output and a historical prompt for the given workspace.
func SetPanelView(workspace string, live bool) {
	ws := normalizeWorkspaceName(workspace)
	panelView.Lock()
	defer panelView.Unlock()
	panelView.byWorkspace[ws] = live
}

// PanelShowsLive defaults to true so a workspace the frontend has never reported
// on still streams to the panel.
func PanelShowsLive(workspace string) bool {
	ws := normalizeWorkspaceName(workspace)
	panelView.Lock()
	defer panelView.Unlock()
	live, ok := panelView.byWorkspace[ws]
	return !ok || live
}

// emitLifecycle reports whether job start/finish events should be emitted. These
// carry the run identity the frontend needs to order the stream, so
// they are never suppressed for a historical view — dropping one would leave the
// frontend unable to render the rest of that run.
func (ctx SessionLogContext) emitLifecycle() bool {
	return ctx.WorkspaceActive && ctx.Emit != nil
}

// emitContent reports whether streamed output should reach the panel.
func (ctx SessionLogContext) emitContent() bool {
	return ctx.emitLifecycle() && PanelShowsLive(ctx.Workspace)
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

	writeSessionOutput(&b, outputBlock)

	b.WriteString("### Stream Trace\n\n<!-- stream trace: start -->\n")
	return b.String()
}

func writeSessionOutput(b *strings.Builder, outputBlock string) {
	trimmed := strings.TrimSpace(head(outputBlock))
	if trimmed == "" {
		return
	}

	var textLines []string
	var imageLines []string
	for _, line := range strings.Split(trimmed, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "![") && strings.Contains(line, "](") && strings.HasSuffix(line, ")") {
			imageLines = append(imageLines, line)
			continue
		}
		textLines = append(textLines, line)
	}

	if len(textLines) > 0 {
		b.WriteString("### Output\n\n~~~text\n")
		b.WriteString(strings.Join(textLines, "\n"))
		b.WriteString("\n~~~\n\n")
	}
	for _, image := range imageLines {
		b.WriteString(image)
		b.WriteString("\n\n")
	}
}

const maxLines = 20

func head(s string) string {
	slice := strings.SplitN(s, "\n", maxLines+1)
	if len(slice) > maxLines {
		slice = slice[:maxLines]
	}
	return strings.Join(slice, "\n")
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

	state = &sessionLogState{
		workspace: ws,
		sessionID: sessionID,
	}
	aiSessionLogStore.byWorkspace[ws] = state
	return state, nil
}

type ActiveStreamIdentity struct {
	SessionID int64
	RunID     uint64
}

func GetActiveStreamIdentity(workspace string) ActiveStreamIdentity {
	ws := normalizeWorkspaceName(workspace)

	aiSessionLogStore.Lock()
	defer aiSessionLogStore.Unlock()

	state := aiSessionLogStore.byWorkspace[ws]
	if state == nil || !state.active {
		return ActiveStreamIdentity{}
	}

	return ActiveStreamIdentity{
		SessionID: state.sessionID,
		RunID:     state.runID,
	}
}

// PromptLogMeta describes a prompt in the active session for the AI panel's
// Prompt dropdown.
type PromptLogMeta struct {
	SessionID int64  `json:"sessionId"`
	PromptID  int64  `json:"promptId"`
	Heading   string `json:"heading"`
	Started   string `json:"started"`
	SizeBytes int64  `json:"sizeBytes"`
	Status    string `json:"status"`
}

// ListPromptLogs returns per-prompt log metadata for the active session in the
// given workspace, ordered chronologically.
func ListPromptLogs(workspace string) []PromptLogMeta {
	ws := normalizeWorkspaceName(workspace)
	sessionID, err := ActiveSessionID(ws)
	if err != nil || sessionID <= 0 {
		return nil
	}
	metas, err := ListStreamPromptMetas(ws, sessionID)
	if err != nil {
		log.Printf("cannot list AI stream prompts for workspace %q: %v", ws, err)
		return nil
	}
	return metas
}

// ClearActiveSessionLog removes the active session's SQLite stream data.
func ClearActiveSessionLog(workspace string) error {
	ws := normalizeWorkspaceName(workspace)
	sessionID, err := ActiveSessionID(ws)
	if err != nil {
		return err
	}
	return ClearSessionLog(ws, sessionID)
}

// ClearSessionLog removes SQLite stream data for the given session.
func ClearSessionLog(workspace string, sessionID int64) error {
	aiSessionLogStore.Lock()
	defer aiSessionLogStore.Unlock()

	ws := normalizeWorkspaceName(workspace)
	if state := aiSessionLogStore.byWorkspace[ws]; state != nil && state.sessionID == sessionID {
		delete(aiSessionLogStore.byWorkspace, ws)
	}
	return deleteStreamSession(ws, sessionID)
}

// DeleteSessionLog is used when the session itself is deleted.
func DeleteSessionLog(workspace string, sessionID int64) error {
	return ClearSessionLog(workspace, sessionID)
}

// DeletePromptLog removes one prompt's SQLite stream data.
func DeletePromptLog(workspace string, sessionID, promptID int64) error {
	return deleteStreamPrompt(workspace, sessionID, promptID)
}

func WriteToSessionLog(ctx SessionLogContext, state int, payload string) {
	workspace := normalizeWorkspaceName(ctx.Workspace)

	switch state {
	case SESSION_LOG_START_JOB:
		now := time.Now()
		var prefix string
		var runID uint64
		var sessionID int64
		aiSessionLogStore.Lock()
		if ref, err := activeSessionLogStateLocked(workspace); err == nil && ref != nil {
			sessionID = ref.sessionID
			if sessionID > 0 {
				allocatedRunID, allocErr := AllocateStreamRunID(workspace, formatSessionLogTimestamp(now))
				if allocErr != nil {
					log.Printf("cannot allocate AI stream run ID for workspace %q: %v", workspace, allocErr)
				} else {
					ref.runID = allocatedRunID
					ref.promptID = 0
					ref.status = PromptStatusRunning
					ref.active = true
					runID = ref.runID
					prefix = buildSessionLogRequestPrefix(ctx.Query, ctx.CommandLine, ctx.OutputBlock, now)
				}
			}
		}
		aiSessionLogStore.Unlock()
		if runID > 0 && sessionID > 0 {
			_ = CreateStreamPrompt(workspace, sessionID, 0, runID, summarizeRequestHeading(ctx.Query), formatSessionLogTimestamp(now))
		}

		if ctx.emitLifecycle() {
			ctx.Emit("aiJobStart", AIJobStart{RunID: runID, Title: prefix})
		}

	case SESSION_LOG_FINALIZE_JOB:
		now := time.Now()
		var sessionID int64
		var runID uint64
		status := ctx.Status
		if status == "" {
			status = PromptStatusSuccess
		}
		aiSessionLogStore.Lock()
		if ref, err := activeSessionLogStateLocked(workspace); err == nil && ref != nil {
			sessionID = ref.sessionID
			runID = ref.runID
			ref.promptID = ctx.PromptID
			ref.status = status
		}
		aiSessionLogStore.Unlock()
		if sessionID > 0 && runID > 0 && ctx.PromptID > 0 {
			if err := FinalizeStreamPrompt(workspace, sessionID, runID, ctx.PromptID, formatSessionLogTimestamp(now), status); err != nil {
				log.Printf("cannot finalize AI stream prompt %d for workspace %q: %v", ctx.PromptID, workspace, err)
			}
		}

	case SESSION_LOG_FINISH_JOB:
		var finish AIJobFinish
		streaming := false
		aiSessionLogStore.Lock()
		if ref, err := activeSessionLogStateLocked(workspace); err == nil && ref != nil {
			if ctx.Status == PromptStatusSuccess || ctx.Status == PromptStatusFailed || ctx.Status == PromptStatusCancelled {
				ref.status = ctx.Status
			}
			finish = AIJobFinish{RunID: ref.runID, PromptID: ref.promptID, Status: ref.status}
			ref.active = false
			streaming = true
		}
		aiSessionLogStore.Unlock()
		if streaming && ctx.emitLifecycle() {
			ctx.Emit("aiJobFinish", finish)
		}
	}
}
