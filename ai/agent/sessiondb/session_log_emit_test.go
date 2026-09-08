package sessiondb

import "testing"

// Job lifecycle events carry the run id and final sequence the frontend needs to
// order a stream. Suppressing one leaves the frontend unable to render the rest
// of that run, even after the user returns to live output.
func TestEmitGating_LifecycleSurvivesHistoricalView(t *testing.T) {
	t.Cleanup(func() { SetPanelView(true) })

	ctx := SessionLogContext{
		WorkspaceActive: true,
		Emit:            func(string, any) {},
	}

	SetPanelView(false)
	if !ctx.emitLifecycle() {
		t.Fatal("emitLifecycle() = false while viewing history, want true")
	}
	if ctx.emitContent() {
		t.Fatal("emitContent() = true while viewing history, want false")
	}

	SetPanelView(true)
	if !ctx.emitLifecycle() || !ctx.emitContent() {
		t.Fatal("live output should emit both lifecycle and content")
	}
}

func TestEmitGating_InactiveWorkspaceEmitsNothing(t *testing.T) {
	t.Cleanup(func() { SetPanelView(true) })
	SetPanelView(true)

	ctx := SessionLogContext{
		WorkspaceActive: false,
		Emit:            func(string, any) {},
	}

	if ctx.emitLifecycle() || ctx.emitContent() {
		t.Fatal("an inactive workspace must not emit to the panel")
	}
}

func TestEmitGating_RequiresEmitter(t *testing.T) {
	t.Cleanup(func() { SetPanelView(true) })
	SetPanelView(true)

	ctx := SessionLogContext{WorkspaceActive: true}
	if ctx.emitLifecycle() || ctx.emitContent() {
		t.Fatal("a context without an emitter must not report emittable")
	}
}
