package tmux

import (
	"strings"
	"testing"
	"time"

	"github.com/lmorg/ttyphoon/types"
	runebuf "github.com/lmorg/ttyphoon/utils/rune_buf"
)

type pendingPaneOutputTestRenderer struct {
	types.Renderer
	refresh func()
}

func (renderer *pendingPaneOutputTestRenderer) RefreshWindowList() {
	renderer.refresh()
}

func TestPendingPaneOutputRefreshesLayoutAfterInitialization(t *testing.T) {
	tmux := &Tmux{panes: newPaneMap()}
	const paneID = "%42"
	if !tmux.queuePaneOutput(paneID, []byte("prompt")) {
		t.Fatal("new pane did not start initialization")
	}
	pane := &PaneT{id: paneID, buf: runebuf.New()}
	t.Cleanup(pane.buf.Close)
	tmux.panes.Set(paneID, pane)
	var refreshes int
	tmux.renderer = &pendingPaneOutputTestRenderer{refresh: func() {
		refreshes++
		if !tmux.pendingPaneOutputMu.TryLock() {
			t.Fatal("layout refresh holds the pending-output lock")
		}
		_, pending := tmux.pendingPaneOutput[paneID]
		tmux.pendingPaneOutputMu.Unlock()
		if pending || pane.buf.BufSize() != len("prompt") {
			t.Fatal("layout refresh ran before pending output was flushed")
		}
	}}
	tmux.finishPendingPaneOutput(paneID, pane, nil)
	if refreshes != 1 {
		t.Fatalf("initialized pane requested %d layout refreshes, want 1", refreshes)
	}
	if tmux.queuePaneOutput(paneID, []byte("more output")) || refreshes != 1 {
		t.Fatal("ordinary output repeated pane initialization or layout refresh")
	}
}

func TestPendingPaneOutputPreservesSplitAnsiSequenceOrder(t *testing.T) {
	tmux := &Tmux{
		panes:             newPaneMap(),
		pendingPaneOutput: make(map[string][][]byte),
		renderer:          &pendingPaneOutputTestRenderer{refresh: func() {}},
	}
	const paneId = "%7"

	chunks := []string{
		"\x1b[38;2;198;208;245",
		"mHello",
		"\x1b[0m",
	}

	if initialize := tmux.queuePaneOutput(paneId, []byte(chunks[0])); !initialize {
		t.Fatal("expected first unknown-pane chunk to start initialization")
	}
	if initialize := tmux.queuePaneOutput(paneId, []byte(chunks[1])); initialize {
		t.Fatal("expected subsequent unknown-pane chunk to join the existing backlog")
	}

	pane := &PaneT{buf: runebuf.New()}
	t.Cleanup(pane.buf.Close)
	tmux.panes.Set(paneId, pane)
	tmux.finishPendingPaneOutput(paneId, pane, nil)

	if initialize := tmux.queuePaneOutput(paneId, []byte(chunks[2])); initialize {
		t.Fatal("expected initialized pane output to use the direct path")
	}

	want := strings.Join(chunks, "")
	result := make(chan string, 1)
	go func() {
		var got strings.Builder
		for range []rune(want) {
			r, err := pane.buf.Read()
			if err != nil {
				result <- ""
				return
			}
			got.WriteRune(r)
		}
		result <- got.String()
	}()

	select {
	case got := <-result:
		if got != want {
			t.Fatalf("pane output order changed: got %q, want %q", got, want)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out reading flushed pane output")
	}
}
