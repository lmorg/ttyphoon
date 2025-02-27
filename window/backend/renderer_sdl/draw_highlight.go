package rendersdl

import (
	"log"

	"github.com/lmorg/mxtty/types"
	"github.com/lmorg/mxtty/window/backend/renderer_sdl/layer"
	"github.com/veandco/go-sdl2/sdl"
)

const (
	highlightAlphaBorder = 190
	highlightAlphaFill   = 128
)

var highlightBlendMode sdl.BlendMode // controlled by LightMode

func (sr *sdlRender) DrawHighlightRect(tileId types.TileId, _topLeftCell, _bottomRightCell *types.XY) {
	topLeftCell := types.XY{
		X: _topLeftCell.X + sr.termWin.Tiles[tileId].TopLeft.X,
		Y: _topLeftCell.Y + sr.termWin.Tiles[tileId].TopLeft.Y,
	}

	bottomRightCell := types.XY{
		X: _bottomRightCell.X + sr.termWin.Tiles[tileId].TopLeft.X,
		Y: _bottomRightCell.Y + sr.termWin.Tiles[tileId].TopLeft.Y,
	}

	sr._drawHighlightRect(
		&sdl.Rect{
			X: (topLeftCell.X * sr.glyphSize.X) + _PANE_LEFT_MARGIN,
			Y: (topLeftCell.Y * sr.glyphSize.Y) + _PANE_TOP_MARGIN,
			W: (bottomRightCell.X * sr.glyphSize.X),
			H: (bottomRightCell.Y * sr.glyphSize.Y),
		},
		highlightBorder, highlightFill,
		highlightAlphaBorder, highlightAlphaFill)
}

func (sr *sdlRender) DrawRectWithColour(tileId types.TileId, _topLeftCell, _bottomRightCell *types.XY, colour *types.Colour, incLeftMargin bool) {
	topLeftCell := types.XY{
		X: _topLeftCell.X + sr.termWin.Tiles[tileId].TopLeft.X,
		Y: _topLeftCell.Y + sr.termWin.Tiles[tileId].TopLeft.Y,
	}

	bottomRightCell := types.XY{
		X: _bottomRightCell.X + sr.termWin.Tiles[tileId].TopLeft.X,
		Y: _bottomRightCell.Y + sr.termWin.Tiles[tileId].TopLeft.Y,
	}

	leftMargin := _PANE_LEFT_MARGIN
	if incLeftMargin {
		leftMargin = _PANE_LEFT_MARGIN_OUTER
	}

	sr._drawHighlightRect(
		&sdl.Rect{
			X: (topLeftCell.X * sr.glyphSize.X) + leftMargin,
			Y: (topLeftCell.Y * sr.glyphSize.Y) + _PANE_TOP_MARGIN,
			W: (bottomRightCell.X * sr.glyphSize.X) + _PANE_LEFT_MARGIN - leftMargin,
			H: (bottomRightCell.Y * sr.glyphSize.Y),
		},
		colour, colour,
		highlightAlphaBorder, highlightAlphaFill)
}

func (sr *sdlRender) _drawHighlightRect(rect *sdl.Rect, colourBorder, colourFill *types.Colour, alphaBorder, alphaFill byte) {
	texture := sr.createRendererTexture()
	if texture == nil {
		return
	}
	defer sr.renderer.SetRenderTarget(nil)

	err := texture.SetBlendMode(highlightBlendMode)
	if err != nil {
		log.Printf("ERROR: %v", err)
	}

	_ = sr.renderer.SetDrawColor(colourBorder.Red, colourBorder.Green, colourBorder.Blue, alphaBorder)
	rect.X -= 1
	rect.Y -= 1
	rect.W += 2
	rect.H += 2

	_ = sr.renderer.DrawRect(rect)
	rect.X += 1
	rect.Y += 1
	rect.W -= 2
	rect.H -= 2
	_ = sr.renderer.DrawRect(rect)

	// fill background

	_ = sr.renderer.SetDrawColor(colourFill.Red, colourFill.Green, colourFill.Blue, alphaFill)
	rect.X += 1
	rect.Y += 1
	rect.W -= 2
	rect.H -= 2
	sr.renderer.FillRect(rect)

	sr.AddToOverlayStack(&layer.RenderStackT{texture, nil, nil, true})
}
