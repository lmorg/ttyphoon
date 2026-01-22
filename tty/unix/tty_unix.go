//go:build !windows
// +build !windows

package tty_unix

import (
	"fmt"
	"io"
	"log"
	"os"

	"github.com/creack/pty"
	"github.com/lmorg/ttyphoon/types"
	runebuf "github.com/lmorg/ttyphoon/utils/rune_buf"
	"golang.org/x/sys/unix"
)

type Pty struct {
	primary   *os.File
	secondary *os.File
	buf       *runebuf.Buf
	process   *os.Process
}

func NewPty(size *types.XY) (types.Pty, error) {
	secondary, primary, err := pty.Open()
	if err != nil {
		return nil, fmt.Errorf("unable to open pty: %s", err.Error())
	}

	err = pty.Setsize(primary, &pty.Winsize{
		Cols: uint16(size.X),
		Rows: uint16(size.Y),
	})
	if err != nil {
		return nil, fmt.Errorf("unable to set pty size: %s", err.Error())
	}

	p := &Pty{
		primary:   primary,
		secondary: secondary,
		buf:       runebuf.New(),
	}

	go p.read(secondary)

	return p, err
}

func OpenPty(path string) (types.Pty, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	p := &Pty{
		primary:   file,
		secondary: file,
		buf:       runebuf.New(),
	}

	go p.read(file)

	return p, nil
}

func (p *Pty) Write(b []byte) error {
	_, err := p.secondary.Write(b)
	return err
}

func (p *Pty) read(f *os.File) {
	for {
		b := make([]byte, 10*1024)
		i, err := f.Read(b)
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
	err := pty.Setsize(p.primary, &pty.Winsize{
		Cols: uint16(size.X),
		Rows: uint16(size.Y),
	})
	if err != nil {
		return err
	}

	if p.process != nil {
		err = p.process.Signal(unix.SIGWINCH)
		if err != nil {
			return err
		}
	}

	return nil
}

func (p *Pty) BufSize() int {
	return p.buf.BufSize()
}

func (p *Pty) Close() {
	p.buf.Close()
	_ = p.primary.Close()   // we don't really care about errors here
	_ = p.secondary.Close() // we don't really care about errors here
}
