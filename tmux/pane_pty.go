package tmux

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/lmorg/mxtty/codes"
	"github.com/lmorg/mxtty/debug"
	"github.com/lmorg/mxtty/types"
	"github.com/lmorg/mxtty/utils/octal"
)

func (p *PaneT) File() *os.File      { return nil }
func (p *PaneT) Read() (rune, error) { return p.buf.Read() }
func (p *PaneT) BufSize() int        { return p.buf.BufSize() }

func (p *PaneT) exit() {
	p.tmux.renderer.DisplayNotification(types.NOTIFY_INFO, fmt.Sprintf("Closing term %s: %s", p.id, p.title))

	p.buf.Close()
	//p.term.Close()
	p.term = nil
	p.Close()

	p.tmux.panes.Delete(p.id)
	win, ok := p.tmux.wins[p.windowId]
	if ok {
		win.panes.Delete(p.id)
	}

	debug.Log(p)
}

func (p *PaneT) Write(b []byte) error {
	if len(b) == 0 {
		return errors.New("nothing to write")
	}

	ok, err := p._hotkey(b)
	if ok {
		if err != nil {
			p.tmux.renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
		}
		return nil
	}

	var flags string
	if b[0] == 0 {
		b = []byte(codes.TmuxKeySanitiser(b))
	} else {
		flags = "-l"
		b = octal.Escape(b)
	}

	command := []byte(fmt.Sprintf(`send-keys %s -t %s `, flags, p.id))
	command = append(command, b...)
	_, err = p.tmux.SendCommand(command)
	return err
}

func (p *PaneT) _hotkey(b []byte) (bool, error) {
	key := codes.TmuxKeySanitiser(b)

	if p.tmux.prefixTtl.Before(time.Now()) {
		if key != p.tmux.keys.prefix {
			// standard key, do nothing
			return false, nil
		}

		// prefix key pressed
		p.tmux.prefixTtl = time.Now().Add(2000 * time.Millisecond)
		return true, nil
	}

	// run tmux function
	fn, ok := p.tmux.keys.fnTable[key]
	if !ok {
		// no function to run, lets treat as standard key
		p.tmux.prefixTtl = time.Now()
		return false, nil
	}

	// valid prefix key, so lets set a repeat key timer
	p.tmux.prefixTtl = time.Now().Add(500 * time.Millisecond)
	return true, fn.fn(p.tmux)
}

func (p *PaneT) Resize(size *types.XY) error {
	command := fmt.Sprintf("resize-pane -t %s -x %d -y %d", p.id, size.X, size.Y)
	_, err := p.tmux.SendCommand([]byte(command))
	p.width = int(size.X)
	p.height = int(size.Y)

	//if err != nil {
	return err
	//}

	//return p.tmux.RefreshClient(size)
}
