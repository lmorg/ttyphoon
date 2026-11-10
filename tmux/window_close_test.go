package tmux

import (
	"testing"
	"time"

	"github.com/lmorg/ttyphoon/types"
)

type windowCloseTestRenderer struct {
	types.Renderer
	refreshed chan struct{}
	errors    chan string
}

func (r *windowCloseTestRenderer) RefreshWindowList() {
	r.refreshed <- struct{}{}
}

func (r *windowCloseTestRenderer) DisplayNotification(level types.NotificationType, message string) {
	if level == types.NOTIFY_ERROR {
		r.errors <- message
	}
}

func TestRespWindowClose(t *testing.T) {
	renderer := &windowCloseTestRenderer{
		refreshed: make(chan struct{}, 1),
		errors:    make(chan string, 1),
	}
	tmux := &Tmux{wins: newWindowMap(), renderer: renderer}
	closed := &WindowT{id: "@2", name: "closed"}
	surviving := &WindowT{id: "@1", name: "surviving"}
	tmux.wins.Set(closed.id, closed)
	tmux.wins.Set(surviving.id, surviving)

	_respWindowClose(tmux, []byte("%window-close @2"))

	select {
	case <-renderer.refreshed:
	case <-time.After(time.Second):
		t.Fatal("window-close did not refresh the window list")
	}
	if !closed.closed || tmux.wins.Get(closed.id) != nil {
		t.Fatal("window-close did not remove the specified window")
	}
	if surviving.closed || tmux.wins.Get(surviving.id) != surviving {
		t.Fatal("window-close changed another window")
	}
	select {
	case message := <-renderer.errors:
		t.Fatalf("unexpected error: %s", message)
	default:
	}
}

func TestRespWindowCloseRejectsMalformedNotification(t *testing.T) {
	for _, input := range []string{"", "%window-close", "%window-close ", "%window-close @2 unexpected"} {
		t.Run(input, func(t *testing.T) {
			renderer := &windowCloseTestRenderer{
				refreshed: make(chan struct{}, 1),
				errors:    make(chan string, 1),
			}
			tmux := &Tmux{wins: newWindowMap(), renderer: renderer}
			_respWindowClose(tmux, []byte(input))
			select {
			case <-renderer.errors:
			default:
				t.Fatal("malformed notification did not report an error")
			}
		})
	}
}
