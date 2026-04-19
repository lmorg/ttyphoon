package virtualterm

import (
	"bytes"
	"io"
	"testing"

	"github.com/lmorg/ttyphoon/types"
)

type parserTestPty struct {
	in  []rune
	out []byte
	res []*types.XY
}

func newParserTestPty() *parserTestPty {
	return &parserTestPty{}
}

func (p *parserTestPty) ExecuteShell(func()) {}
func (p *parserTestPty) Read() (rune, error) {
	if len(p.in) == 0 {
		return 0, io.EOF
	}

	r := p.in[0]
	p.in = p.in[1:]
	return r, nil
}
func (p *parserTestPty) Write(b []byte) error {
	p.out = append(p.out, b...)
	return nil
}
func (p *parserTestPty) Resize(size *types.XY) error {
	if size == nil {
		p.res = append(p.res, nil)
		return nil
	}

	p.res = append(p.res, &types.XY{X: size.X, Y: size.Y})
	return nil
}
func (p *parserTestPty) BufSize() int { return len(p.in) }
func (p *parserTestPty) Close()       {}

func (p *parserTestPty) FeedInput(b []byte) {
	p.in = append(p.in, bytes.Runes(b)...)
}

func newParserTestTerm() *Term {
	size := &types.XY{X: _testTermWidth, Y: _testTermHeight}
	pty := newParserTestPty()

	term := &Term{
		size: size,
		sgr:  types.SGR_DEFAULT.Copy(),
		Pty:  pty,
	}

	term._blockMeta = NewRowBlockMeta(term)
	term._normBuf = term.makeScreen()
	term.screen = &term._normBuf
	term.setJumpScroll()

	return term
}

func drainMockPtyInput(t *testing.T, term *Term, maxReads int) {
	t.Helper()

	reads := 0
	for term.Pty.BufSize() > 0 {
		if reads >= maxReads {
			t.Fatalf("parser did not drain input within %d reads", maxReads)
		}

		r, err := term.Pty.Read()
		if err != nil {
			t.Fatalf("unable to read from mock PTY: %v", err)
		}
		term.readChar(r)
		reads++
	}
}

func TestLookupSgr_ParsesMixedUnderlineAndTrueColor(t *testing.T) {
	sgr := types.SGR_DEFAULT.Copy()

	lookupSgr(sgr, []int32{4, 38, 2, 166, 226, 46})

	if !sgr.Bitwise.Is(types.SGR_UNDERLINE) {
		t.Fatalf("expected underline to be set")
	}

	if sgr.Fg.Red != 166 || sgr.Fg.Green != 226 || sgr.Fg.Blue != 46 {
		t.Fatalf("unexpected foreground colour: got (%d,%d,%d)", sgr.Fg.Red, sgr.Fg.Green, sgr.Fg.Blue)
	}
}

func TestReadChar_ParsesMixedSgrSequenceFromStream(t *testing.T) {
	term := newParserTestTerm()
	pty := term.Pty.(*parserTestPty)

	stream := "\x1b[4;38;2;166;226;46mX"
	pty.FeedInput([]byte(stream))

	drainMockPtyInput(t, term, len(stream)*2)

	cell := (*term.screen)[0].Cells[0]
	if cell.Char != 'X' {
		t.Fatalf("expected first visible char to be X, got %q", cell.Char)
	}

	if !cell.Sgr.Bitwise.Is(types.SGR_UNDERLINE) {
		t.Fatalf("expected rendered cell to be underlined")
	}

	if cell.Sgr.Fg.Red != 166 || cell.Sgr.Fg.Green != 226 || cell.Sgr.Fg.Blue != 46 {
		t.Fatalf("unexpected rendered foreground colour: got (%d,%d,%d)", cell.Sgr.Fg.Red, cell.Sgr.Fg.Green, cell.Sgr.Fg.Blue)
	}
}

func TestReadChar_ConsumesBatTerminalQueryReplies(t *testing.T) {
	term := newParserTestTerm()
	pty := term.Pty.(*parserTestPty)

	stream := "\x1b]10;rgb:ca/d3/f5\a\x1b]11;rgb:24/27/3a\a\x1b[?65;1;6;15;17;22;28;29c"
	pty.FeedInput([]byte(stream))

	drainMockPtyInput(t, term, len(stream)*2)

	for y := range *term.screen {
		for x := range (*term.screen)[y].Cells {
			if char := (*term.screen)[y].Cells[x].Char; char != 0 {
				t.Fatalf("expected escape-only stream to not render visible characters, found %q at row %d col %d", char, y, x)
			}
		}
	}
}

func TestReadChar_ConsumesPrivateCsiDaResponseInStream(t *testing.T) {
	term := newParserTestTerm()
	pty := term.Pty.(*parserTestPty)

	stream := "A\x1b[?65;1;6;15;17;22;28;29cB"
	pty.FeedInput([]byte(stream))

	drainMockPtyInput(t, term, len(stream)*2)

	first := (*term.screen)[0].Cells[0].Char
	second := (*term.screen)[0].Cells[1].Char
	if first != 'A' || second != 'B' {
		t.Fatalf("expected visible text to remain around private CSI response, got %q %q", first, second)
	}
}
