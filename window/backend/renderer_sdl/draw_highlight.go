package rendersdl

import (
	"log"

	"github.com/lmorg/ttyphoon/debug"
	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/window/backend/renderer_sdl/layer"
	"github.com/veandco/go-sdl2/sdl"
)

const (
	highlightAlphaBorder = 190
	highlightAlphaFill   = 128
)

var highlightBlendMode sdl.BlendMode // controlled by LightMode

func (sr *sdlRender) DrawHighlightRect(tile types.Tile, _topLeftCell, bottomRightCell *types.XY) {
	topLeftCell := types.XY{
		X: _topLeftCell.X + tile.Left(),
		Y: _topLeftCell.Y + tile.Top(),
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

func (sr *sdlRender) DrawRectWithColour(tile types.Tile, _topLeftCell, _bottomRightCell *types.XY, colour *types.Colour, incLeftMargin bool) {
	debug.Log(_topLeftCell)
	debug.Log(_bottomRightCell)

	topLeftCell := types.XY{
		X: _topLeftCell.X,
		Y: max(_topLeftCell.Y, 0),
	}

	bottomRightCell := types.XY{
		X: _bottomRightCell.X,
		Y: _bottomRightCell.Y + min(_topLeftCell.Y, 0),
	}

	if bottomRightCell.Y+topLeftCell.Y > tile.GetTerm().GetSize().Y {
		bottomRightCell.Y = tile.GetTerm().GetSize().Y - topLeftCell.Y
	}

	leftMargin := _PANE_LEFT_MARGIN
	if incLeftMargin {
		leftMargin = _PANE_LEFT_MARGIN_OUTER
	}

	sr._drawHighlightRect(
		&sdl.Rect{
			X: ((topLeftCell.X + tile.Left()) * sr.glyphSize.X) + leftMargin,
			Y: ((topLeftCell.Y + tile.Top()) * sr.glyphSize.Y) + _PANE_TOP_MARGIN,
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
