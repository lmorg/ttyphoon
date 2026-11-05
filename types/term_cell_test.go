package types

import "testing"

type cellTestElement struct {
	Element
	position XY
}

func (element *cellTestElement) Rune(pos *XY) rune {
	element.position = *pos
	return 'E'
}

func TestCellElementModes(t *testing.T) {
	element := &cellTestElement{}
	drawable := &Cell{Char: SetElementXY(&XY{X: 2, Y: 3}), Element: element}
	if !drawable.HasDrawableElement() || drawable.HasTextElement() || drawable.Rune() != 'E' ||
		element.position != (XY{X: 2, Y: 3}) {
		t.Fatal("drawable element lost its encoded coordinates or rune delegation")
	}

	text := &Cell{Char: '界', Element: &TextElement{Element: element}}
	if text.HasDrawableElement() || !text.HasTextElement() || text.Rune() != '界' ||
		*text.GetElementXY() != (XY{}) {
		t.Fatal("text element did not preserve the character or used it as coordinates")
	}
	text.Char = 0
	if text.Rune() != ' ' {
		t.Fatal("text element did not preserve blank-cell semantics")
	}
	text.Clear()
	if text.Element != nil || text.HasTextElement() || text.HasDrawableElement() {
		t.Fatal("clearing a cell retained its element mode")
	}

	plain := &Cell{Char: 'P'}
	if plain.HasDrawableElement() || plain.HasTextElement() || plain.Rune() != 'P' {
		t.Fatal("plain-cell behaviour changed")
	}
}

func TestCellGetSetElementXY(t *testing.T) {
	tests := []*XY{
		{0, 0},
		{1, 1},
		{3, 7},
		{7, 3},
		{200, 0},
		{0, 200},
		{200, 200},
		{10000, 13},
		{13, 10000},
		{10000, 10000},
		{32767, 1},
		{1, 32767},
		{32767, 32767},
	}

	for i, expected := range tests {
		cell := new(Cell)
		cell.Char = SetElementXY(expected)
		actual := cell.GetElementXY()
		if expected.X != actual.X || expected.Y != actual.Y {
			t.Errorf("Mismatch in test %d", i)
			t.Logf("Expected: X: %d, Y: %d", expected.X, expected.Y)
			t.Logf("Actual:   X: %d, Y: %d", actual.X, actual.Y)
		}
	}
}
