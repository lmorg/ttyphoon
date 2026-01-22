package virtualterm

import (
	"bytes"
	"strings"
	"testing"

	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/types"
)

const _testTermHeight = 5
const _testTermWidth = 10

// NewTestTerminal creates a new virtual term used for unit tests
func NewTestTerminal() *Term {
	config.Config.Tmux.Enabled = false
	
	size := &types.XY{X: _testTermWidth, Y: _testTermHeight}

	term := &Term{}

	term.reset(size)

	return term
}

type testCondition struct {
	Screen    string
	Operation func(t *testing.T, term *Term)
	Expected  string
}

type testTerm struct {
	Tests     []testCondition
	Operation func(t *testing.T, term *Term)
}

func (tt *testTerm) RunTests(t *testing.T) {
	t.Helper()

	for i, test := range tt.Tests {
		term := NewTestTerminal()
		term.setJumpScroll()

		for _, r := range test.Screen {
			if r == '\n' {
				continue
			}
			term.readChar(r)
		}

		begin := term.exportAsString()

		if tt.Operation != nil {
			tt.Operation(t, term)
		}

		if test.Operation != nil {
			test.Operation(t, term)
		}

		expected := _pad(test.Expected)
		actual := strings.ReplaceAll(term.exportAsString(), "·", ".")
		//actual := term.exportAsString()
		if actual != expected {
			t.Errorf("Expected doesn't match Actual in test %d:", i)
			t.Logf("  Raw Begin: '%s'", strings.ReplaceAll(begin, "\n", "↲"))
			t.Logf("  Raw End:   '%s'", strings.ReplaceAll(actual, "\n", "↲"))
			t.Logf("  Screen:%s", _indent(_pad(test.Screen)))
			t.Logf("  Expected:%s", _indent(expected))
			t.Logf("  Actual:%s", _indent(actual))
		}
	}
}

var _padding = bytes.Repeat(
	append(bytes.Repeat([]byte{'.'}, _testTermWidth), '\n'),
	_testTermHeight)

func _pad(s string) string {
	padded := bytes.Clone(_padding)
	copy(padded, s)
	return string(padded)
}

func _indent(s string) string {
	s = strings.ReplaceAll("\n"+s[:len(s)-1], "\n", "|\n    |")
	return s[1:] + "|"
}

// writeCellsTest is not a practical function for the terminal however it does
// provide a helpful wrapper for unit tests.
func (term *Term) writeCells(s string) {
	for _, r := range s {
		term.writeCell(r, nil)
	}
}

func (term *Term) exportAsString() string {
	//term._mutex.Lock()
	//defer term._mutex.Unlock()

	var (
		r = make([]rune, (term.size.X+1)*term.size.Y)
		i int
	)

	for y := range *term.screen {
		for x := range (*term.screen)[y].Cells {
			if (*term.screen)[y].Cells[x].Char > 0 {
				r[i] = (*term.screen)[y].Cells[x].Char
			} else {
				r[i] = '·'
			}
			i++
		}
		r[i] = '\n'
		i++
	}

	return string(r)
}
