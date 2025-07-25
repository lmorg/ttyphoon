package rendersdl

import (
	"github.com/lmorg/mxtty/types"
	"github.com/lmorg/mxtty/window/backend/renderer_sdl/layer"
	"github.com/veandco/go-sdl2/sdl"
)

func (sr *sdlRender) DrawOutputBlockChrome(tile types.Tile, _start, n int32, c *types.Colour, folded bool) {
	if _start >= tile.GetTerm().GetSize().Y {
		return
	}

	start := _start + tile.Top()

	texture := sr.createRendererTexture()
	if texture == nil {
		return
	}
	defer sr.renderer.SetRenderTarget(nil)
	defer sr.AddToElementStack(&layer.RenderStackT{texture, nil, nil, true})

	height := n
	if _start+n >= tile.GetTerm().GetSize().Y {
		height = tile.GetTerm().GetSize().Y - _start - 1
	}

	rect := &sdl.Rect{
		X: (tile.Left() * sr.glyphSize.X) + _PANE_LEFT_MARGIN_OUTER,
		Y: (start * sr.glyphSize.Y) + _PANE_TOP_MARGIN,
		W: _PANE_BLOCK_HIGHLIGHT,
		H: (height + 1) /*n*/ * sr.glyphSize.Y,
	}

	if folded {
		rect.W = _PANE_BLOCK_FOLDED
	}

	_ = sr.renderer.SetDrawColor(c.Red, c.Green, c.Blue, 192)
	//_ = texture.SetBlendMode(sdl.BLENDMODE_ADD)
	_ = sr.renderer.FillRect(rect)

	if !folded && start+n <= tile.Bottom() {
		x2 := (tile.Right()+2)*sr.glyphSize.X + _PANE_LEFT_MARGIN_OUTER - 1
		_ = sr.renderer.DrawLine(rect.X, rect.Y+rect.H, x2, rect.Y+rect.H)
	}
}

func (sr *sdlRender) DrawScrollbar(tile types.Tile, value, max int) {
	f := float64(value) / float64(max)

	rect := &sdl.Rect{
		X: (tile.Right()+1)*sr.glyphSize.X + _PANE_LEFT_MARGIN_OUTER - (sr.glyphSize.X / 2),
		Y: (tile.Top()+1)*sr.glyphSize.Y + _PANE_TOP_MARGIN - (sr.glyphSize.Y / 2),
		W: sr.glyphSize.X,
		H: (tile.GetTerm().GetSize().Y - 1) * sr.glyphSize.Y,
	}

	c := &types.Colour{Red: 128, Green: 128, Blue: 128}
	sr._drawHighlightRect(rect, c, c, 128, 32)

	texture := sr.createRendererTexture()
	if texture == nil {
		return
	}
	rect.H = int32(float64(rect.H) * f)
	_ = sr.renderer.SetDrawColor(c.Red, c.Green, c.Blue, 192)
	_ = texture.SetBlendMode(sdl.BLENDMODE_ADD)
	_ = sr.renderer.FillRect(rect)

	defer sr.renderer.SetRenderTarget(nil)
	defer sr.AddToOverlayStack(&layer.RenderStackT{texture, nil, nil, true})
}
