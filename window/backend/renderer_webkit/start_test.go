package rendererwebkit

import "testing"

func newPauseTestRenderer() *webkitRender {
	return &webkitRender{_redraw: make(chan struct{}, 1)}
}

func TestRedrawSkipsRenderingWhilePaused(t *testing.T) {
	wr := newPauseTestRenderer()
	wr.SetRenderingPaused(true)
	wr.enqueueDrawCommand(DrawCommand{Op: DrawOpTileOverlay})

	emitted := 0
	wr.redraw(func(uint64, []DrawOpTuple) { emitted++ })

	if emitted != 0 {
		t.Fatalf("redraw emitted %d batches while paused", emitted)
	}
	if len(wr.drawCommands) != 1 {
		t.Fatalf("paused redraw consumed queued commands: %d left", len(wr.drawCommands))
	}
}

func TestRedrawEmitsAfterResume(t *testing.T) {
	wr := newPauseTestRenderer()
	wr.SetRenderingPaused(true)
	wr.enqueueDrawCommand(DrawCommand{Op: DrawOpTileOverlay})
	wr.SetRenderingPaused(false)

	select {
	case <-wr._redraw:
	default:
		t.Fatal("resuming rendering did not request a redraw")
	}

	emitted := 0
	var batchID uint64
	wr.redraw(func(id uint64, commands []DrawOpTuple) {
		emitted++
		batchID = id
		if len(commands) == 0 {
			t.Fatal("emitted an empty batch")
		}
	})
	if emitted != 1 {
		t.Fatalf("expected 1 emitted batch after resume, got %d", emitted)
	}
	if batchID == 0 {
		t.Fatal("emitted batch has no ID")
	}
}

func TestRedrawWaitsForAcknowledgement(t *testing.T) {
	wr := newPauseTestRenderer()
	wr.enqueueDrawCommand(DrawCommand{Op: DrawOpTileOverlay})

	emitted := 0
	var firstBatchID uint64
	var batchID uint64
	emit := func(id uint64, _ []DrawOpTuple) {
		emitted++
		if emitted == 1 {
			firstBatchID = id
		}
		batchID = id
	}
	wr.redraw(emit)
	if emitted != 1 {
		t.Fatalf("expected first batch to emit, got %d", emitted)
	}

	wr.enqueueDrawCommand(DrawCommand{Op: DrawOpTileOverlay})
	wr.redraw(emit)
	if emitted != 1 {
		t.Fatalf("emitted another batch before acknowledgement: %d", emitted)
	}
	if len(wr.drawCommands) != 1 {
		t.Fatalf("redraw consumed pending commands while waiting for acknowledgement: %d left", len(wr.drawCommands))
	}

	wr.AcknowledgeRedraw(batchID)
	select {
	case <-wr._redraw:
	default:
		t.Fatal("acknowledgement did not request a redraw")
	}
	wr.redraw(emit)
	if emitted != 2 {
		t.Fatalf("expected next batch after acknowledgement, got %d", emitted)
	}
	if batchID <= firstBatchID {
		t.Fatalf("batch IDs did not increase: first %d, next %d", firstBatchID, batchID)
	}

	wr.AcknowledgeRedraw(firstBatchID)
	wr.redraw(emit)
	if emitted != 2 {
		t.Fatalf("stale acknowledgement released the next batch: emitted %d", emitted)
	}
}

func TestAcknowledgementWithoutDeferredRedrawDoesNotLoop(t *testing.T) {
	wr := newPauseTestRenderer()
	wr.enqueueDrawCommand(DrawCommand{Op: DrawOpTileOverlay})
	var batchID uint64
	wr.redraw(func(id uint64, _ []DrawOpTuple) { batchID = id })

	wr.AcknowledgeRedraw(batchID)
	select {
	case <-wr._redraw:
		t.Fatal("acknowledgement requested a redraw although none was held back")
	default:
	}
}

func TestRedrawIgnoresStaleAcknowledgement(t *testing.T) {
	wr := newPauseTestRenderer()
	wr.enqueueDrawCommand(DrawCommand{Op: DrawOpTileOverlay})
	emitted := 0
	wr.redraw(func(uint64, []DrawOpTuple) { emitted++ })

	wr.AcknowledgeRedraw(999)
	wr.redraw(func(uint64, []DrawOpTuple) { emitted++ })
	if emitted != 1 {
		t.Fatalf("stale acknowledgement released backpressure: emitted %d batches", emitted)
	}
}

func TestSetRenderingPausedIgnoresRepeatedState(t *testing.T) {
	wr := newPauseTestRenderer()

	wr.SetRenderingPaused(false)
	select {
	case <-wr._redraw:
		t.Fatal("resuming an already running renderer requested a redraw")
	default:
	}

	wr.SetRenderingPaused(true)
	wr.SetRenderingPaused(true)
	select {
	case <-wr._redraw:
		t.Fatal("pausing requested a redraw")
	default:
	}
}
