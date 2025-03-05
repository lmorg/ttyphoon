package rendersdl

import (
	"github.com/lmorg/mxtty/config"
	"github.com/lmorg/mxtty/types"
	"github.com/veandco/go-sdl2/sdl"
)

func (sr *sdlRender) _isLigaturePair(cells []*types.Cell) int {
	if len(cells) < 2 || cells[0] == nil || cells[1] == nil ||
		cells[0].Char == 0 || cells[1].Char == 0 ||
		cells[0].Char == ' ' || cells[1].Char == ' ' {
		return -1
	}

	for id, lig := range config.LigaturePairs() {
		if lig[0] != cells[0].Char {
			continue
		}
		if lig[1] != cells[1].Char {
			continue
		}
		if cells[0].Sgr.HashValue() == cells[1].Sgr.HashValue() {
			return id
		}
	}

	return -1
}

func cellsToRunes(cells []*types.Cell) []rune {
	r := make([]rune, len(cells))
	for i := range cells {
		r[i] = cells[i].Char
	}
	return r
}

func (sr *sdlRender) PrintLigature(tile *types.Tile, cells []*types.Cell, _cellPos *types.XY, ligId int) {
	cellPos := types.XY{
		X: _cellPos.X + tile.Left,
		Y: _cellPos.Y + tile.Left,
	}
	glyphSizeX := sr.glyphSize.X * 2

	dstRect := &sdl.Rect{
		X: (sr.glyphSize.X * cellPos.X) + _PANE_LEFT_MARGIN,
		Y: (sr.glyphSize.Y * cellPos.Y) + _PANE_TOP_MARGIN,
		W: glyphSizeX + dropShadowOffset,
		H: sr.glyphSize.Y + dropShadowOffset,
	}

	hlTexture := _HLTEXTURE_NONE
	if cells[0].Sgr.Bitwise.Is(types.SGR_HIGHLIGHT_SEARCH_RESULT) {
		hlTexture = _HLTEXTURE_SEARCH_RESULT
	}
	if isCellHighlighted(sr, dstRect) {
		hlTexture = _HLTEXTURE_SELECTION
	}

	hash := cells[0].Sgr.HashValue()

	ok := sr.fontCache.atlas.Render(sr, dstRect, rune(ligId), hash, hlTexture)
	if ok {
		return
	}

	extAtlases, ok := sr.fontCache.ligs[rune(ligId)]
	if ok {
		for i := range extAtlases {
			ok = extAtlases[i].Render(sr, dstRect, rune(ligId), hash, hlTexture)
			if ok {
				return
			}
		}
	}

	atlas := newFontAtlas(cellsToRunes(cells), cells[0].Sgr, &types.XY{X: glyphSizeX, Y: sr.glyphSize.Y}, sr.renderer, sr.font, rune(ligId))
	sr.fontCache.ligs[rune(ligId)] = append(sr.fontCache.ligs[rune(ligId)], atlas)
	if !atlas.Render(sr, dstRect, rune(ligId), hash, hlTexture) {
		panic("failed")
	}
}
