package tmux

import (
	"strings"
	"testing"
	"time"

	runebuf "github.com/lmorg/ttyphoon/utils/rune_buf"
)

func TestPendingPaneOutputPreservesSplitAnsiSequenceOrder(t *testing.T) {
	tmux := &Tmux{
		panes:             newPaneMap(),
		pendingPaneOutput: make(map[string][][]byte),
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
