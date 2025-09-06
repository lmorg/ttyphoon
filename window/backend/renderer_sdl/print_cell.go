package rendersdl

import (
	"unsafe"

	"github.com/lmorg/mxtty/config"
	"github.com/lmorg/mxtty/debug"
	"github.com/lmorg/mxtty/types"
	"github.com/lmorg/mxtty/utils/runewidth"
	"github.com/veandco/go-sdl2/sdl"
)

const dropShadowOffset int32 = 1

const (
	_HLTEXTURE_NONE          = iota
	_HLTEXTURE_SELECTION     // should always be first non-zero value
	_HLTEXTURE_SEARCH_RESULT //
	_HLTEXTURE_HEADING       //
	_HLTEXTURE_MATCH_RANGE   //
	_HLTEXTURE_LAST          // placeholder for rect calculations. Must always come last
)

var textShadow = []*types.Colour{ // RGBA
	_HLTEXTURE_NONE:          types.COLOR_TEXT_SHADOW,
	_HLTEXTURE_SELECTION:     types.COLOR_SELECTION,
	_HLTEXTURE_SEARCH_RESULT: types.COLOR_SEARCH_RESULT,
	_HLTEXTURE_HEADING:       {255, 0, 0, 128},
	_HLTEXTURE_MATCH_RANGE:   {64, 255, 64, 255},
}

func sgrOpts(sgr *types.Sgr, forceBg bool) (fg *types.Colour, bg *types.Colour) {
	if sgr.Bitwise.Is(types.SGR_INVERT) {
		bg, fg = sgr.Fg, sgr.Bg
	} else {
		fg, bg = sgr.Fg, sgr.Bg
	}

	if unsafe.Pointer(bg) == unsafe.Pointer(types.SGR_COLOR_BACKGROUND) && !forceBg {
		bg = nil
	}

	return fg, bg
}

func (sr *sdlRender) PrintCell(tile types.Tile, cell *types.Cell, _cellPos *types.XY) {
	if cell.Char == 0 || _cellPos.X < 0 || _cellPos.Y < 0 {
		return
	}
	if cell.Sgr == nil {
		if debug.Enabled {
			panic("ERROR: nil sgr")
		} else {
			sr.DisplayNotification(types.NOTIFY_DEBUG, "ERROR: nil sgr")
		}
		return
	}

	if tile.GetTerm() != nil {
		tileSize := tile.GetTerm().GetSize()
		if _cellPos.X >= tileSize.X || _cellPos.Y >= tileSize.Y {
			return
		}
	}

	sr.printCell(tile, cell, _cellPos)
}

func (sr *sdlRender) printCell(tile types.Tile, cell *types.Cell, _cellPos *types.XY) {
	cellPos := types.XY{
		X: _cellPos.X + tile.Left(),
		Y: _cellPos.Y + tile.Top(),
	}

	glyphSizeX := sr.glyphSize.X

	if cell.Sgr.Bitwise.Is(types.SGR_WIDE_CHAR) {
		glyphSizeX *= 2
	}

	dstRect := &sdl.Rect{
		X: (sr.glyphSize.X * cellPos.X) + _PANE_LEFT_MARGIN,
		Y: (sr.glyphSize.Y * cellPos.Y) + _PANE_TOP_MARGIN,
		W: glyphSizeX + dropShadowOffset,
		H: sr.glyphSize.Y + dropShadowOffset,
	}

	hlTexture := _HLTEXTURE_NONE
	if cell.Sgr.Bitwise.Is(types.SGR_HIGHLIGHT_SEARCH_RESULT) {
		hlTexture = _HLTEXTURE_SEARCH_RESULT
	}
	if isCellHighlighted(sr, dstRect) {
		hlTexture = _HLTEXTURE_SELECTION
	}

	hash := cell.Sgr.HashValue()

	ok := sr.fontCache.atlas.Render(sr, dstRect, cell.Char, hash, hlTexture)
	if ok {
		return
	}

	extAtlases, ok := sr.fontCache.extended[cell.Char]
	if ok {
		for i := range extAtlases {
			ok = extAtlases[i].Render(sr, dstRect, cell.Char, hash, hlTexture)
			if ok {
				return
			}
		}
	}

	atlas := newFontAtlas([]rune{cell.Char}, cell.Sgr, &types.XY{X: glyphSizeX, Y: sr.glyphSize.Y}, sr.renderer, _FONT_ATLAS_NOT_LIG)
	sr.fontCache.extended[cell.Char] = append(sr.fontCache.extended[cell.Char], atlas)
	atlas.Render(sr, dstRect, cell.Char, hash, hlTexture)
}

func (sr *sdlRender) printCellRect(ch rune, sgr *types.Sgr, dstRect *sdl.Rect) {
	glyphSizeX := sr.glyphSize.X * int32(runewidth.RuneWidth(ch))
	if sgr.Bitwise.Is(types.SGR_WIDE_CHAR) {
		glyphSizeX *= 2
	}

	hlTexture := _HLTEXTURE_NONE
	switch {
	case sgr.Bitwise.Is(types.SGR_HIGHLIGHT_HEADING):
		hlTexture = _HLTEXTURE_HEADING

	case isCellHighlighted(sr, dstRect):
		hlTexture = _HLTEXTURE_SELECTION

	case sgr.Bitwise.Is(types.SGR_HIGHLIGHT_SEARCH_RESULT):
		hlTexture = _HLTEXTURE_SEARCH_RESULT
	}

	hash := sgr.HashValue()

	ok := sr.fontCache.atlas.RenderAsOverlay(sr, dstRect, ch, hash, hlTexture)
	if ok {
		return
	}

	extAtlases, ok := sr.fontCache.extended[ch]
	if ok {
		for i := range extAtlases {
			ok = extAtlases[i].RenderAsOverlay(sr, dstRect, ch, hash, hlTexture)
			if ok {
				return
			}
		}
	}

	atlas := newFontAtlas([]rune{ch}, sgr, &types.XY{X: glyphSizeX, Y: sr.glyphSize.Y}, sr.renderer, _FONT_ATLAS_NOT_LIG)
	sr.fontCache.extended[ch] = append(sr.fontCache.extended[ch], atlas)
	atlas.RenderAsOverlay(sr, dstRect, ch, hash, hlTexture)
}

func (sr *sdlRender) printString(s string, sgr *types.Sgr, pos *types.XY) {
	rect := &sdl.Rect{
		X: pos.X,
		Y: pos.Y,
		//W: sr.glyphSize.X + dropShadowOffset,
		H: sr.glyphSize.Y + dropShadowOffset,
	}

	for _, r := range s {
		rect.W = int32(runewidth.RuneWidth(r)) * (sr.glyphSize.X + dropShadowOffset)
		sr.printCellRect(r, sgr, rect)
		rect.X += rect.W - 1
	}
}

func (sr *sdlRender) PrintRow(tile types.Tile, cells []*types.Cell, _cellPos *types.XY) {
	l := int32(len(cells))

	if tile.GetTerm() != nil && tile.GetTerm().GetSize().X <= _cellPos.X+l {
		l = tile.GetTerm().GetSize().X - _cellPos.X
	}

	cellPos := &types.XY{
		X: _cellPos.X,
		Y: _cellPos.Y,
	}

	for ; cellPos.X < l; cellPos.X++ {
		if cells[cellPos.X] == nil {
			continue
		}

		if config.Config.TypeFace.Ligatures {
			ligId := sr._isLigaturePair(cells[cellPos.X:])
			if ligId >= 0 {
				sr.PrintLigature(tile, cells[cellPos.X:cellPos.X+2], cellPos, ligId)
				cellPos.X++
				continue
			}
		}
		sr.printCell(tile, cells[cellPos.X], cellPos)
	}
}
