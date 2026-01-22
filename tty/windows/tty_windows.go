//go:build ignore
// +build ignore

package tty_windows

import (
	"io"
	"log"

	"github.com/lmorg/ttyphoon/types"
	runebuf "github.com/lmorg/ttyphoon/utils/rune_buf"
	conpty "github.com/qsocket/conpty-go"
)

type Pty struct {
	h, w   int
	conPty *conpty.ConPty
	buf    *runebuf.Buf
}

func NewPty(size *types.XY) (types.Pty, error) {
	p := &Pty{
		h:   int(size.X),
		w:   int(size.Y),
		buf: runebuf.New(),
	}

	return p, nil
}

func (p *Pty) Write(b []byte) error {
	_, err := p.conPty.Write(b)
	return err
}

func (p *Pty) read() {
	for {
		b := make([]byte, 10*1024)
		i, err := p.conPty.Read(b)
		if err != nil && err.Error() != io.EOF.Error() {
			log.Printf("ERROR: problem reading from Pty (%d bytes dropped): %v", i, err)
			continue
		}

		p.buf.Write(b[:i])
	}
}

func (p *Pty) Read() (rune, error) {
	return p.buf.Read()
}

func (p *Pty) Resize(size *types.XY) error {
	return p.conPty.Resize(int(size.X), int(size.Y))
}

func (p *Pty) BufSize() int {
	return p.buf.BufSize()
}

func (p *Pty) Close() {
	p.buf.Close()
	_ = p.conPty.Close() // we don't really care about errors here
}
