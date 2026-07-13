package ai

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/lmorg/ttyphoon/utils/notes"
)

//var aiWorkspaceFilenameSanitizer = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

type aiSessionState struct {
	loaded       bool
	completed    string
	activePrefix string
	activeText   string
}

var aiSessionCacheStore = struct {
	sync.Mutex
	byWorkspace map[string]*aiSessionState
}{
	byWorkspace: map[string]*aiSessionState{},
}

func normalizeWorkspaceName(workspace string) string {
	return workspace
	/*w := strings.TrimSpace(workspace)
	if w == "" {
		return "default"
	}

	w = aiWorkspaceFilenameSanitizer.ReplaceAllString(w, "_")
	w = strings.Trim(w, "._-")
	if w == "" {
		return "default"
	}

	return w*/
}

func aiSessionCachePath(workspace string) (string, error) {
	ws := normalizeWorkspaceName(workspace)
	// Reuse Notes directory mapping so AI cache tracks the same app-level
	// path conventions used by Notes/history.
	base := filepath.Clean(notes.DirGlobal())
	dir := filepath.Dir(base)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	return filepath.Join(dir, fmt.Sprintf("ai.session.%s.md", ws)), nil
}

func loadWorkspaceSession(workspace string) *aiSessionState {
	ws := normalizeWorkspaceName(workspace)

	state := aiSessionCacheStore.byWorkspace[ws]
	if state == nil {
		state = &aiSessionState{}
		aiSessionCacheStore.byWorkspace[ws] = state
	}

	if state.loaded {
		return state
	}

	path, err := aiSessionCachePath(ws)
	if err != nil {
		state.loaded = true
		return state
	}

	b, err := os.ReadFile(path)
	if err == nil {
		state.completed = string(b)
	}
	state.loaded = true

	return state
}

func (s *aiSessionState) merged() string {
	return s.completed + s.activePrefix + s.activeText
}

func persistWorkspaceSession(workspace string, state *aiSessionState) {
	path, err := aiSessionCachePath(workspace)
	if err != nil {
		return
	}

	_ = os.WriteFile(path, []byte(state.merged()), 0o644)
}

func GetSessionCache(workspace string) string {
	aiSessionCacheStore.Lock()
	defer aiSessionCacheStore.Unlock()

	state := loadWorkspaceSession(workspace)
	return state.merged()
}

func beginActiveJobLocked(state *aiSessionState, title string) {
	safeTitle := strings.TrimSpace(title)
	if safeTitle == "" {
		safeTitle = "AI response"
	}

	ts := time.Now().Format("15:04:05")
	state.activePrefix = fmt.Sprintf("\n\n### [%s] %s\n\n", ts, safeTitle)
	state.activeText = ""
}

func SessionCacheStartJob(workspace, title string) {
	aiSessionCacheStore.Lock()
	defer aiSessionCacheStore.Unlock()

	ws := normalizeWorkspaceName(workspace)
	state := loadWorkspaceSession(ws)

	if state.activePrefix != "" || state.activeText != "" {
		state.completed += state.activePrefix + state.activeText + "\n\n"
	}

	beginActiveJobLocked(state, title)
	persistWorkspaceSession(ws, state)
}

func SessionCacheAppendChunk(workspace, chunk string) {
	if chunk == "" {
		return
	}

	aiSessionCacheStore.Lock()
	defer aiSessionCacheStore.Unlock()

	ws := normalizeWorkspaceName(workspace)
	state := loadWorkspaceSession(ws)

	if state.activePrefix == "" {
		beginActiveJobLocked(state, "AI response")
	}

	state.activeText += chunk
	persistWorkspaceSession(ws, state)
}

func SessionCacheFinalizeJob(workspace, finalOutput string) {
	aiSessionCacheStore.Lock()
	defer aiSessionCacheStore.Unlock()

	ws := normalizeWorkspaceName(workspace)
	state := loadWorkspaceSession(ws)

	if state.activePrefix == "" {
		beginActiveJobLocked(state, "AI response")
	}

	state.activeText = finalOutput
	state.completed += state.activePrefix + state.activeText + "\n\n"
	state.activePrefix = ""
	state.activeText = ""
	persistWorkspaceSession(ws, state)
}
