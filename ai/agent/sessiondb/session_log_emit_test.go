package sessiondb

import (
	"os"
	"testing"
)

func resetPanelView(t *testing.T, workspaces ...string) {
	t.Helper()

	t.Cleanup(func() {
		panelView.Lock()
		defer panelView.Unlock()
		for _, ws := range workspaces {
			delete(panelView.byWorkspace, normalizeWorkspaceName(ws))
		}
	})
}

// Job lifecycle events carry the run id and final sequence the frontend needs to
// order a stream. Suppressing one leaves the frontend unable to render the rest
// of that run, even after the user returns to live output.
func TestEmitGating_LifecycleSurvivesHistoricalView(t *testing.T) {
	resetPanelView(t, "alpha")

	ctx := SessionLogContext{
		Workspace:       "alpha",
		WorkspaceActive: true,
		Emit:            func(string, any) {},
	}

	SetPanelView("alpha", false)
	if !ctx.emitLifecycle() {
		t.Fatal("emitLifecycle() = false while viewing history, want true")
	}
	if ctx.emitContent() {
		t.Fatal("emitContent() = true while viewing history, want false")
	}

	SetPanelView("alpha", true)
	if !ctx.emitLifecycle() || !ctx.emitContent() {
		t.Fatal("live output should emit both lifecycle and content")
	}
}

// PanelShowsLive is exported because the typed block transport gates on it
// directly, without going through a SessionLogContext.
func TestPanelShowsLive_TracksPerWorkspaceView(t *testing.T) {
	resetPanelView(t, "alpha", "beta")

	if !PanelShowsLive("workspace-never-reported") {
		t.Fatal("PanelShowsLive() = false for an unreported workspace, want true")
	}

	SetPanelView("alpha", false)
	SetPanelView("beta", true)

	if PanelShowsLive("alpha") {
		t.Fatal("PanelShowsLive(alpha) = true while showing history, want false")
	}
	if !PanelShowsLive("beta") {
		t.Fatal("PanelShowsLive(beta) = false while live, want true")
	}
}

func TestWriteToSessionLog_DoesNotCreateMarkdownFile(t *testing.T) {
	workspace := "stream-only-log-test"
	path, err := dbPath(workspace)
	if err != nil {
		t.Fatalf("dbPath: %v", err)
	}
	pendingPath, err := sessionLogPendingPath(workspace, 1)
	if err != nil {
		t.Fatalf("sessionLogPendingPath: %v", err)
	}
	_ = os.Remove(path)
	_ = os.Remove(pendingPath)
	t.Cleanup(func() {
		_ = os.Remove(path)
		_ = os.Remove(pendingPath)
	})

	WriteToSessionLog(SessionLogContext{Workspace: workspace, Query: "query"}, SESSION_LOG_START_JOB, "")
	if _, err := os.Stat(pendingPath); !os.IsNotExist(err) {
		t.Fatalf("pending markdown file exists after start: %v", err)
	}
}

func TestEmitGating_InactiveWorkspaceEmitsNothing(t *testing.T) {
	resetPanelView(t, "alpha")
	SetPanelView("alpha", true)

	ctx := SessionLogContext{
		Workspace:       "alpha",
		WorkspaceActive: false,
		Emit:            func(string, any) {},
	}

	if ctx.emitLifecycle() || ctx.emitContent() {
		t.Fatal("an inactive workspace must not emit to the panel")
	}
}

func TestEmitGating_RequiresEmitter(t *testing.T) {
	resetPanelView(t, "alpha")
	SetPanelView("alpha", true)

	ctx := SessionLogContext{Workspace: "alpha", WorkspaceActive: true}
	if ctx.emitLifecycle() || ctx.emitContent() {
		t.Fatal("a context without an emitter must not report emittable")
	}
}

// Concurrent agents run in separate workspaces, so reading history in one must
// never gag a live run in another.
func TestEmitGating_PanelViewIsPerWorkspace(t *testing.T) {
	resetPanelView(t, "alpha", "beta")

	SetPanelView("alpha", false)
	SetPanelView("beta", true)

	alpha := SessionLogContext{Workspace: "alpha", WorkspaceActive: true, Emit: func(string, any) {}}
	beta := SessionLogContext{Workspace: "beta", WorkspaceActive: true, Emit: func(string, any) {}}

	if alpha.emitContent() {
		t.Fatal("alpha is showing history, want emitContent() = false")
	}
	if !beta.emitContent() {
		t.Fatal("beta is live, want emitContent() = true despite alpha showing history")
	}
}

// A workspace the frontend has never reported on must still stream, otherwise
// the very first run of a session would render nothing.
func TestEmitGating_UnknownWorkspaceDefaultsToLive(t *testing.T) {
	ctx := SessionLogContext{
		Workspace:       "workspace-never-reported",
		WorkspaceActive: true,
		Emit:            func(string, any) {},
	}

	if !ctx.emitContent() {
		t.Fatal("emitContent() = false for an unreported workspace, want true")
	}
}
