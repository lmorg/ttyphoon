package tmux

import (
	"errors"
	"fmt"
	"os"

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
	win := p.tmux.wins.Get(p.windowId)
	if win != nil {
		win.panes.Delete(p.id)
	}

	debug.Log(p)
}

func (p *PaneT) Write(b []byte) error {
	if len(b) == 0 {
		return errors.New("nothing to write")
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
	_, err := p.tmux.SendCommand(command)
	return err
}

func (p *PaneT) Resize(size *types.XY) error {
	command := fmt.Sprintf("resize-pane -t %s -x %d -y %d", p.id, size.X, size.Y)
	_, err := p.tmux.SendCommand([]byte(command))
	p.width = int(size.X)
	p.height = int(size.Y)

	return err
}

func (p *PaneT) ExecuteShell(func()) {}
