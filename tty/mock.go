package tty

import (
	"os"

	"github.com/lmorg/ttyphoon/debug"
	"github.com/lmorg/ttyphoon/types"
	runebuf "github.com/lmorg/ttyphoon/utils/rune_buf"
)

type MockPty struct{ buf *runebuf.Buf }

func (p *MockPty) File() *os.File {
	debug.Log(nil)
	return nil
}

func (p *MockPty) Write(b []byte) error {
	debug.Log(b)
	p.buf.Write(b)
	return nil
}

func (p *MockPty) Read() (rune, error) {
	//debug.Log("read")
	return p.buf.Read()
}

func (p *MockPty) Resize(size *types.XY) error {
	debug.Log(size)
	return nil
}

func (p *MockPty) BufSize() int {
	return p.buf.BufSize()
}

func (p *MockPty) Close() {
	debug.Log(nil)
	p.buf.Close()
}

func NewMock() types.Pty {
	return &MockPty{buf: runebuf.New()}
}

func (p *MockPty) ExecuteShell(_ func()) {}
