package virtualterm

import (
	"errors"
	"testing"

	"github.com/lmorg/ttyphoon/types"
)

type textElementTestElement struct {
	types.Element
	written []rune
	draws   int
	fail    bool
}

func (element *textElementTestElement) Write(r rune) error {
	if element.fail {
		return errors.New("element write failed")
	}
	element.written = append(element.written, r)
	return nil
}

func (element *textElementTestElement) Draw(*types.XY)                 { element.draws++ }
func (element *textElementTestElement) Rune(*types.XY) rune            { return 'D' }
func (element *textElementTestElement) Generate(*types.ApcSlice) error { return nil }
func (element *textElementTestElement) Size() *types.XY                { return &types.XY{X: 1, Y: 1} }

type textElementTestRenderer struct {
	oscHyperlinkTestRenderer
	rows          [][]*types.Cell
	notifications []string
}

func (renderer *textElementTestRenderer) PrintRow(_ types.Tile, cells []*types.Cell, _ *types.XY) {
	renderer.rows = append(renderer.rows, append([]*types.Cell(nil), cells...))
}

func (renderer *textElementTestRenderer) DisplayNotification(_ types.NotificationType, message string) {
	renderer.notifications = append(renderer.notifications, message)
}

func TestTextElementPreservesTextAlongsideCapturedAndDrawableElements(t *testing.T) {
	term := newParserTestTerm()
	renderer := &textElementTestRenderer{}
	term.renderer = renderer
	text := &textElementTestElement{}
	term._activeTextElement = &types.TextElement{Element: text}

	captured := &textElementTestElement{}
	term._activeElement = captured
	term.writeCell('C', nil)
	if string(captured.written) != "C" || len(text.written) != 0 || term._curPos.X != 0 {
		t.Fatal("mxAPC capture no longer takes precedence over text annotation")
	}
	term._activeElement = nil
	term.writeCell('T', nil)
	drawable := &textElementTestElement{}
	term._mxapcGenerate(drawable, types.NewApcSliceNoParse(nil))
	term._activeTextElement = nil
	term.writeCell('P', nil)
	cells := (*term.screen)[0].Cells
	if string(text.written) != "T" || !cells[0].HasTextElement() || cells[0].Rune() != 'T' ||
		cells[1].Element != drawable || !cells[1].HasDrawableElement() || cells[1].Rune() != 'D' ||
		cells[2].Element != nil || cells[2].Rune() != 'P' {
		t.Fatal("text, drawable, and ordinary cells were not kept distinct")
	}

	term._renderCells(*term.screen)
	if text.draws != 0 || drawable.draws != 1 || len(renderer.cells) != 2 ||
		renderer.cells[0].Char != 'T' || renderer.cells[1].Char != 'P' {
		t.Fatal("cell rendering did not preserve the two element modes")
	}
	term._renderLigs(*term.screen)
	if text.draws != 0 || drawable.draws != 2 || renderer.rows[0][0] != cells[0] ||
		renderer.rows[0][1] != nil || renderer.rows[0][2] != cells[2] {
		t.Fatal("ligature rendering did not preserve the two element modes")
	}
}

func TestTextElementResumesAfterMxAPCEnd(t *testing.T) {
	term := newParserTestTerm()
	term.renderer = &textElementTestRenderer{}
	text := &textElementTestElement{}
	term._activeTextElement = &types.TextElement{Element: text}
	captured := &textElementTestElement{}
	term._activeElement = captured
	term.writeCell('C', nil)
	term.mxapcEnd(types.NewApcSliceNoParse(nil))
	term.writeCell('T', nil)
	cells := (*term.screen)[0].Cells
	if term._activeElement != nil || cells[0].Element != captured || !cells[0].HasDrawableElement() ||
		!cells[1].HasTextElement() || cells[1].Rune() != 'T' ||
		string(captured.written) != "C" || string(text.written) != "T" {
		t.Fatal("mxAPC end did not place the captured element and resume normal text annotation")
	}
}

func TestTextElementWriteErrorRetainsTextAndReportsFailure(t *testing.T) {
	term := newParserTestTerm()
	renderer := &textElementTestRenderer{}
	term.renderer = renderer
	term._activeTextElement = &types.TextElement{Element: &textElementTestElement{fail: true}}
	term.writeCell('A', nil)
	if term._activeTextElement != nil || (*term.screen)[0].Cells[0].Element != nil ||
		(*term.screen)[0].Cells[0].Rune() != 'A' || len(renderer.notifications) != 1 {
		t.Fatal("failed text element lost text or failed to report and clear itself")
	}
}
