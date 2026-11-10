package tmux

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/lmorg/ttyphoon/types"
	runebuf "github.com/lmorg/ttyphoon/utils/rune_buf"
)

type pendingPaneOutputTestRenderer struct {
	types.Renderer
	refresh func()
	notify  func(string)
}

func (renderer *pendingPaneOutputTestRenderer) RefreshWindowList() {
	renderer.refresh()
}

func (renderer *pendingPaneOutputTestRenderer) DisplayNotification(_ types.NotificationType, message string) {
	if renderer.notify != nil {
		renderer.notify(message)
	}
}

func TestPendingPaneOutputSurvivesFailedInitializationAndFlushesOnDiscovery(t *testing.T) {
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = reader.Close()
		_ = writer.Close()
	})

	tmux := &Tmux{
		tty:               writer,
		resp:              make(chan *tmuxResponseT, 1),
		wins:              newWindowMap(),
		panes:             newPaneMap(),
		pendingPaneOutput: make(map[string]pendingPaneOutputT),
	}
	const paneID = "%42"
	win := &WindowT{id: "@1", panes: newPaneMap()}
	tmux.wins.Set(win.id, win)

	var notification string
	tmux.renderer = &pendingPaneOutputTestRenderer{
		refresh: func() {},
		notify:  func(message string) { notification = message },
	}

	if !tmux.queuePaneOutput(paneID, []byte("early ")) {
		t.Fatal("first output did not start pane initialization")
	}
	tmux.finishPendingPaneOutput(paneID, nil, fmt.Errorf("temporary list-panes failure"))
	if notification != "temporary list-panes failure" {
		t.Fatalf("initialization failure notification = %q", notification)
	}
	if got := tmux.pendingPaneOutput[paneID].chunks; len(got) != 1 || string(got[0]) != "early " {
		t.Fatalf("failed initialization discarded queued output: %q", got)
	}
	if !tmux.queuePaneOutput(paneID, []byte("output")) {
		t.Fatal("output after failure did not schedule another initialization attempt")
	}

	pane := &PaneT{id: paneID, windowId: win.id, width: 80, height: 24, tmux: tmux, buf: runebuf.New()}
	t.Cleanup(pane.buf.Close)
	tmux.panes.Set(paneID, pane)
	win.panes.Set(paneID, pane)

	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		if scanner := bufio.NewScanner(reader); scanner.Scan() {
			tmux.resp <- &tmuxResponseT{Message: [][]byte{[]byte(
				"Id:%42|||Width:80|||Height:24|||PosLeft:0|||PosTop:0|||PosRight:80|||PosBottom:24|||Active:false|||Dead:false|||WindowId:@1|||WinActive:true|||AtBottom:false|||Path:/tmp|||PathStart:/tmp|||Zoomed:false",
			)}}
		}
	}()

	if err := tmux.updatePaneInfo(paneID); err != nil {
		t.Fatalf("later pane discovery failed: %v", err)
	}
	_ = writer.Close()
	<-serverDone

	if _, ok := tmux.pendingPaneOutput[paneID]; ok {
		t.Fatal("pending output remained after pane discovery")
	}
	const want = "early output"
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
			t.Fatalf("replayed pane output = %q, want %q", got, want)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out reading replayed pane output")
	}
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
		pendingPaneOutput: make(map[string]pendingPaneOutputT),
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

func TestPaneOutputPreservesOsc8Sequences(t *testing.T) {
	client := &Tmux{panes: newPaneMap()}
	pane := &PaneT{id: "%0", buf: runebuf.New()}
	t.Cleanup(pane.buf.Close)
	client.panes.Set(pane.id, pane)
	for _, notification := range []string{
		`%output %0 \033]8;id=test;https://example.com/a;b\033`,
		`%output %0 \134Label\033]8;;\007Z`,
	} {
		_respOutput(client, []byte(notification))
	}
	const want = "\x1b]8;id=test;https://example.com/a;b\x1b\\Label\x1b]8;;\aZ"
	result := make(chan string, 1)
	go func() {
		var got strings.Builder
		for range []rune(want) {
			r, err := pane.Read()
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
			t.Fatalf("OSC 8 control-mode output changed: got %q, want %q", got, want)
		}
	case <-time.After(time.Second):
		t.Fatal("OSC 8 control-mode output stalled")
	}
}
