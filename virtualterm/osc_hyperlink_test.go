package virtualterm

import (
	"strings"
	"testing"
	"time"

	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/tty"
	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/window/elements/element_hyperlink"
)

type oscHyperlinkTestTile struct {
	types.Tile
	term *Term
}

func (tile oscHyperlinkTestTile) GetTerm() types.Term { return tile.term }

type oscHyperlinkTestRenderer struct {
	synchronizedOutputTestRenderer
	menu       []types.MenuItem
	cells      []types.Cell
	highlights [][2]types.XY
	status     string
}

func (renderer *oscHyperlinkTestRenderer) NewElement(tile types.Tile, id types.ElementID, _ ...any) types.Element {
	if id != types.ELEMENT_ID_HYPERLINK {
		panic("unexpected element type")
	}
	return element_hyperlink.New(renderer, tile)
}

func (renderer *oscHyperlinkTestRenderer) AddToContextMenu(items ...types.MenuItem) {
	renderer.menu = append(renderer.menu, items...)
}
func (renderer *oscHyperlinkTestRenderer) StatusBarText(text string)            { renderer.status = text }
func (renderer *oscHyperlinkTestRenderer) ShowTooltip(string)                   {}
func (renderer *oscHyperlinkTestRenderer) CloseTooltip()                        {}
func (renderer *oscHyperlinkTestRenderer) SetKeyboardFnMode(types.KeyboardMode) {}
func (renderer *oscHyperlinkTestRenderer) PrintCell(_ types.Tile, cell *types.Cell, _ *types.XY) {
	renderer.cells = append(renderer.cells, *cell)
}
func (renderer *oscHyperlinkTestRenderer) DrawHighlightRect(_ types.Tile, pos, size *types.XY) {
	renderer.highlights = append(renderer.highlights, [2]types.XY{*pos, *size})
}

func newOscHyperlinkTestTerm(t *testing.T, enabled bool) (*Term, *oscHyperlinkTestRenderer) {
	t.Helper()
	previous := config.Config.Terminal
	previousWindow := config.Config.Window
	t.Cleanup(func() {
		config.Config.Terminal = previous
		config.Config.Window = previousWindow
	})
	config.Config.Terminal.OSC8Hyperlinks = enabled
	term := newParserTestTerm()
	term._altBuf = term.makeScreen()
	renderer := &oscHyperlinkTestRenderer{}
	term.renderer = renderer
	term.tile = oscHyperlinkTestTile{term: term}
	return term, renderer
}

func feedOscHyperlinkTest(t *testing.T, term *Term, stream string) {
	t.Helper()
	term.Pty.(*parserTestPty).FeedInput([]byte(stream))
	drainMockPtyInput(t, term, len(stream)*2)
}

func TestOsc8OpenCloseReplaceAndParameters(t *testing.T) {
	for _, terminator := range []string{"\a", "\x1b\\"} {
		t.Run(terminator, func(t *testing.T) {
			term, _ := newOscHyperlinkTestTerm(t, true)
			feedOscHyperlinkTest(t, term, "A\x1b]8;id=first:future=value;https://example.com/a;b?q=c;d"+terminator+
				"BC\x1b]8;;mailto:someone@example.com"+terminator+"D\x1b]8;;"+terminator+"E")
			cells := (*term.screen)[0].Cells
			if got := (*term.screen)[0].String(); got != "ABCDE" {
				t.Fatalf("escape sequences changed visible text: %q", got)
			}
			if cells[0].Element != nil || cells[4].Element != nil || term._activeTextElement != nil {
				t.Fatal("hyperlink escaped its open/close boundaries")
			}
			first := cells[1].Element.(*types.TextElement).Element.(*element_hyperlink.ElementHyperlink)
			if cells[1].Element != cells[2].Element || first.Url() != "https://example.com/a;b?q=c;d" || first.Label() != "BC" {
				t.Fatalf("incorrect first hyperlink: %#v", first)
			}
			second := cells[3].Element.(*types.TextElement).Element.(*element_hyperlink.ElementHyperlink)
			if first == second || second.Url() != "mailto:someone@example.com" || second.Label() != "D" {
				t.Fatalf("incorrect replacement hyperlink: %#v", second)
			}
		})
	}
}

func TestOsc8DisabledAndInvalid(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(map[bool]string{false: "disabled", true: "invalid"}[enabled], func(t *testing.T) {
			term, _ := newOscHyperlinkTestTerm(t, enabled)
			feedOscHyperlinkTest(t, term, "\x1b]8;;https://example.com\aA\x1b]8;missing-uri\aB"+
				"\x1b]8;;relative/path\aC\x1b]8;;1bad:value\aD\x1b]8;;\aE")
			for i, cell := range (*term.screen)[0].Cells[:5] {
				if cell.Char != []rune("ABCDE")[i] {
					t.Fatal("disabled or invalid OSC leaked into text")
				}
				if cell.Element != nil && (!enabled || i != 0) {
					t.Fatalf("unexpected hyperlink at column %d", i)
				}
			}
		})
	}
}

func TestOsc8TextLayoutStylesAndHover(t *testing.T) {
	term, renderer := newOscHyperlinkTestTerm(t, true)
	config.Config.Terminal.AutoHyperlink = false
	feedOscHyperlinkTest(t, term, "12345678\x1b]8;;https://example.com\a\x1b[31mAB界\uFE0EC\x1b]8;;\aD")
	first := (*term.screen)[0].Cells[8]
	wide := (*term.screen)[1].Cells[0]
	if !first.HasTextElement() || first.Char != 'A' || wide.Char != '界' || !wide.Sgr.Bitwise.Is(types.SGR_WIDE_CHAR) {
		t.Fatal("OSC 8 did not preserve normal character representation")
	}
	if term._curPos != (types.XY{X: 4, Y: 1}) || (*term.screen)[1].Cells[2].Char != 'C' {
		t.Fatalf("incorrect Unicode/wrapped cursor position: %+v", term._curPos)
	}
	if wide.Element != first.Element || interactiveCell(*term.screen, &types.XY{X: 1, Y: 1}).Element != first.Element {
		t.Fatal("wrapped/wide text lost its hyperlink")
	}
	assertCellForeground(t, term, 8, types.SGR_COLOR_RED)
	if got := first.Element.(*types.TextElement).Element.(*element_hyperlink.ElementHyperlink).Label(); got != "AB界C" {
		t.Fatalf("unexpected hyperlink label %q", got)
	}

	config.Config.Window.HoverEffectHighlight = true
	term.textElementHover(*term.screen, first.Element)()
	if len(renderer.highlights) != 2 || renderer.highlights[0] != ([2]types.XY{{X: 8}, {X: 2, Y: 1}}) ||
		renderer.highlights[1] != ([2]types.XY{{Y: 1}, {X: 3, Y: 1}}) {
		t.Fatalf("incorrect wrapped hyperlink highlight: %+v", renderer.highlights)
	}
	config.Config.Window.HoverEffectHighlight = false
	term._mouseIn = first.Element
	term._renderCells(*term.screen)
	if !renderer.cells[8].Sgr.Bitwise.Is(types.SGR_UNDERLINE) || first.Sgr.Bitwise.Is(types.SGR_UNDERLINE) {
		t.Fatal("hover underline must not mutate stored terminal styles")
	}
	term.MouseLeave()
	if term.textElementRenderCell(first) != first {
		t.Fatal("hover underline remained after leaving")
	}
}

func TestOsc8OverwriteEraseAndAutoDetection(t *testing.T) {
	term, _ := newOscHyperlinkTestTerm(t, true)
	feedOscHyperlinkTest(t, term, "\x1b]8;;https://real.example\ahttp://x\x1b]8;;\a")
	link := (*term.screen)[0].Cells[0].Element
	_autoHyperlinkElement(term, []*types.Row{(*term.screen)[0]}, []int{0, 8}, "http://x", "http://x")
	for _, cell := range (*term.screen)[0].Cells[:8] {
		if cell.Element != link || !cell.HasTextElement() {
			t.Fatal("auto detection replaced an application-provided target")
		}
	}
	feedOscHyperlinkTest(t, term, "\rZ")
	if (*term.screen)[0].Cells[0].Element != nil || (*term.screen)[0].Cells[1].Element != link {
		t.Fatal("overwrite did not remove exactly the overwritten cell's link")
	}
	feedOscHyperlinkTest(t, term, "\x1b[2K")
	for _, cell := range (*term.screen)[0].Cells {
		if cell.Element != nil {
			t.Fatal("erasing text retained a hyperlink")
		}
	}
	feedOscHyperlinkTest(t, term, "\r\x1b]8;;https://example.com\a界\x1b]8;;\a\rZ")
	if (*term.screen)[0].Cells[1].Element != nil {
		t.Fatal("overwriting a wide character retained its trailing hyperlink")
	}
}

func TestOsc8ExistingContextMenuAndHover(t *testing.T) {
	term, renderer := newOscHyperlinkTestTerm(t, true)
	feedOscHyperlinkTest(t, term, "\x1b]8;;https://target.example/path\aLabel\x1b]8;;\a.")
	pos := &types.XY{X: 2}
	callback := false
	term.MouseClick(pos, types.MOUSE_BUTTON_RIGHT, 1, types.BUTTON_RELEASED, func() { callback = true })
	if !callback || len(renderer.menu) == 0 || renderer.menu[1].Title != "Copy link to clipboard" {
		t.Fatal("OSC 8 did not use the existing hyperlink context menu")
	}
	term.MouseMotion(pos, &types.XY{X: 1}, func() { t.Fatal("link motion fell through") })
	if !strings.Contains(renderer.status, "https://target.example/path") {
		t.Fatalf("hover did not show the actual target: %q", renderer.status)
	}
	term.MouseMotion(&types.XY{X: 5}, &types.XY{X: 1}, func() {})
	if term._mouseIn != nil || renderer.status != "" {
		t.Fatal("moving off a hyperlink retained its hover state")
	}
	if got := string(term.CopyRange(&types.XY{}, &types.XY{X: 4})); got != "Label" {
		t.Fatalf("copying linked text returned %q", got)
	}
}

func TestOsc8ScrollbackAndResize(t *testing.T) {
	term, _ := newOscHyperlinkTestTerm(t, true)
	feedOscHyperlinkTest(t, term, "\x1b]8;;https://example.com\aLabel\x1b]8;;\a\r\n2\r\n3\r\n4\r\n5\r\n6")
	if len(term._scrollBuf) != 1 || term._scrollBuf[0].String() != "Label" ||
		term._scrollBuf[0].Cells[0].Element == nil {
		t.Fatal("scrolling lost linked text or its target")
	}

	term, _ = newOscHyperlinkTestTerm(t, true)
	feedOscHyperlinkTest(t, term, "\x1b]8;;https://example.com\aABCDEFGHIJKL\x1b]8;;\a")
	link := (*term.screen)[0].Cells[0].Element
	term.Resize(&types.XY{X: 5, Y: 5})
	for i, r := range "ABCDEFGHIJKL" {
		cell := (*term.screen)[i/5].Cells[i%5]
		if cell.Char != r || cell.Element != link {
			t.Fatalf("reflow changed linked character %d", i)
		}
	}
}

func TestOsc8BufferTransitionsAndReset(t *testing.T) {
	term, _ := newOscHyperlinkTestTerm(t, true)
	feedOscHyperlinkTest(t, term, "\x1b]8;;https://normal.example\aA")
	normalLink := (*term.screen)[0].Cells[0].Element
	term.csiScreenBufferAlternative()
	if term._activeTextElement != nil {
		t.Fatal("open normal-buffer target leaked into alternate screen")
	}
	feedOscHyperlinkTest(t, term, "\x1b]8;;https://alt.example\aB")
	term.csiScreenBufferNormal()
	if term._activeTextElement != nil || (*term.screen)[0].Cells[0].Element != normalLink {
		t.Fatal("buffer transition lost normal text or retained an active target")
	}
	// Reset without a tile avoids creating the application scrollback database.
	term.tile = nil
	term.reset(term.size)
	if term._activeTextElement != nil || (*term.screen)[0].Cells[0].Element != nil {
		t.Fatal("terminal reset retained hyperlinks")
	}
}

func TestOsc8SplitByteStream(t *testing.T) {
	term, _ := newOscHyperlinkTestTerm(t, true)
	pty := tty.NewMock()
	t.Cleanup(pty.Close)
	term.Pty = pty
	stream := "\x1b]8;id=split;https://example.com/界;a\x1b\\A界\x1b]8;;\x1b\\Z"
	finished := make(chan struct{})
	go func() {
		defer close(finished)
		for {
			r, err := pty.Read()
			if err != nil {
				return
			}
			term.readChar(r)
			if term._curPos.X == 4 {
				return
			}
		}
	}()
	for _, b := range []byte(stream) {
		if err := pty.Write([]byte{b}); err != nil {
			t.Fatal(err)
		}
	}
	select {
	case <-finished:
	case <-time.After(2 * time.Second):
		pty.Close()
		<-finished
		t.Fatal("parser stalled on byte-split OSC 8/UTF-8 stream")
	}
	cells := (*term.screen)[0].Cells
	if cells[0].Char != 'A' || cells[1].Char != '界' || cells[3].Char != 'Z' ||
		cells[0].Element == nil || cells[3].Element != nil {
		t.Fatal("byte-split stream lost text or hyperlink boundaries")
	}
}
