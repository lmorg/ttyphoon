package sessiondb

import "testing"

func seedSessionLogState(t *testing.T, workspace string, state *sessionLogState) {
	t.Helper()

	ws := normalizeWorkspaceName(workspace)

	aiSessionLogStore.Lock()
	previous, existed := aiSessionLogStore.byWorkspace[ws]
	aiSessionLogStore.byWorkspace[ws] = state
	aiSessionLogStore.Unlock()

	t.Cleanup(func() {
		aiSessionLogStore.Lock()
		defer aiSessionLogStore.Unlock()
		if existed {
			aiSessionLogStore.byWorkspace[ws] = previous
			return
		}
		delete(aiSessionLogStore.byWorkspace, ws)
	})
}

// The panel resyncs its ordered cursor from this snapshot, so the sequence it
// reports must be the next unsent one and the text everything streamed so far.
func TestGetActiveStreamSnapshot_ReturnsInFlightRun(t *testing.T) {
	state := &sessionLogState{
		workspace:   "alpha",
		sessionID:   7,
		requestOpen: true,
		runID:       3,
		sequence:    12,
	}
	state.streamed.WriteString("partial answer")
	seedSessionLogState(t, "alpha", state)

	got := GetActiveStreamSnapshot("alpha")

	if !got.Active {
		t.Fatal("Active = false, want true for an open request")
	}
	if got.RunID != 3 {
		t.Fatalf("RunID = %d, want 3", got.RunID)
	}
	if got.Sequence != 12 {
		t.Fatalf("Sequence = %d, want 12", got.Sequence)
	}
	if got.Text != "partial answer" {
		t.Fatalf("Text = %q, want %q", got.Text, "partial answer")
	}
}

func TestGetActiveStreamSnapshot_InactiveWhenRequestClosed(t *testing.T) {
	state := &sessionLogState{
		workspace:   "beta",
		sessionID:   9,
		requestOpen: false,
		runID:       4,
		sequence:    5,
	}
	state.streamed.WriteString("finished answer")
	seedSessionLogState(t, "beta", state)

	got := GetActiveStreamSnapshot("beta")

	if got.Active {
		t.Fatal("Active = true for a closed request, want false")
	}
	if got.Text != "" {
		t.Fatalf("Text = %q, want empty for a closed request", got.Text)
	}
}

func TestGetActiveStreamSnapshot_UnknownWorkspaceIsInactive(t *testing.T) {
	got := GetActiveStreamSnapshot("workspace-with-no-session-log-state")

	if got.Active {
		t.Fatal("Active = true for an unknown workspace, want false")
	}
}
