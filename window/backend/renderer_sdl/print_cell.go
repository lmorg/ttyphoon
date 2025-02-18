package rendersdl

import (
	"unsafe"

	"github.com/lmorg/mxtty/config"
	"github.com/lmorg/mxtty/types"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

const dropShadowOffset int32 = 1

const (
	_HLTEXTURE_NONE      = iota
	_HLTEXTURE_SELECTION // should always be first non-zero value
	_HLTEXTURE_SEARCH_RESULT
	_HLTEXTURE_MATCH_RANGE
	_HLTEXTURE_LAST // placeholder for rect calculations. Must always come last
)

var textShadow = []*types.Colour{ // RGBA
	_HLTEXTURE_NONE:          {0, 0, 0, 0},
	_HLTEXTURE_SELECTION:     {64, 64, 255, 255},
	_HLTEXTURE_SEARCH_RESULT: {64, 64, 255, 255},
	_HLTEXTURE_MATCH_RANGE:   {64, 255, 64, 255},
}

func fontStyle(style types.SgrFlag) ttf.Style {
	var i ttf.Style

	if style.Is(types.SGR_BOLD) && !config.Config.TypeFace.Ligatures {
		i |= ttf.STYLE_BOLD
	}

	if style.Is(types.SGR_ITALIC) {
		i |= ttf.STYLE_ITALIC
	}

	if style.Is(types.SGR_UNDERLINE) {
		i |= ttf.STYLE_UNDERLINE
	}

	if style.Is(types.SGR_STRIKETHROUGH) {
		i |= ttf.STYLE_STRIKETHROUGH
	}

	return i
}

func sgrOpts(sgr *types.Sgr, forceBg bool) (fg *types.Colour, bg *types.Colour) {
	if sgr.Bitwise.Is(types.SGR_INVERT) {
		bg, fg = sgr.Fg, sgr.Bg
	} else {
		fg, bg = sgr.Fg, sgr.Bg
	}

	if unsafe.Pointer(bg) == unsafe.Pointer(types.SGR_DEFAULT.Bg) && !forceBg {
		bg = nil
	}

	return fg, bg
}

func (sr *sdlRender) PrintCell(cell *types.Cell, cellPos *types.XY) {
	if cell.Char == 0 {
		return
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
	hash := cell.Sgr.HashValue() // TODO: am i hashing on the right key? Should it be rune?

	ok := sr.fontCache.atlas.Render(sr, dstRect, cell.Char, hash, hlTexture)
	if ok {
		return
	}

	extAtlases, ok := sr.fontCache.extended[hash]
	if ok {
		for i := range extAtlases {
			ok = extAtlases[i].Render(sr, dstRect, cell.Char, hash, hlTexture)
			if ok {
				return
			}
		}
	}

	atlas := newFontAtlas([]rune{cell.Char}, cell.Sgr, &types.XY{X: glyphSizeX, Y: sr.glyphSize.Y}, sr.renderer, sr.font)
	sr.fontCache.extended[hash] = append(sr.fontCache.extended[hash], atlas)
	atlas.Render(sr, dstRect, cell.Char, hash, hlTexture)
}
