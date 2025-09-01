package rendersdl

import (
	"github.com/lmorg/mxtty/types"
	"github.com/lmorg/mxtty/window/backend/renderer_sdl/layer"
	"github.com/veandco/go-sdl2/sdl"
)

func (sr *sdlRender) DrawGaugeH(tile types.Tile, topLeft *types.XY, width int32, value, max int, c *types.Colour) {
	sr.drawGaugeH(&sdl.Rect{
		X: (topLeft.X+tile.Left()+1)*sr.glyphSize.X + _PANE_LEFT_MARGIN_OUTER - (sr.glyphSize.X / 2),
		Y: (topLeft.Y+tile.Top())*sr.glyphSize.Y + _PANE_TOP_MARGIN - (sr.glyphSize.Y / 2),
		W: width * sr.glyphSize.X,
		H: sr.glyphSize.X,
	}, value, max, c)
}

func (sr *sdlRender) drawGaugeH(rect *sdl.Rect, value, max int, c *types.Colour) {
	sr._drawHighlightRect(rect, c, c, 128, 32)

	texture := sr.createRendererTexture()
	if texture == nil {
		return
	}

	rect.W = int32(float64(rect.W) * (float64(value) / float64(max)))
	_ = sr.renderer.SetDrawColor(c.Red, c.Green, c.Blue, 192)
	_ = texture.SetBlendMode(sdl.BLENDMODE_ADD)
	_ = sr.renderer.FillRect(rect)

	sr.renderer.SetRenderTarget(nil)
	sr.AddToOverlayStack(&layer.RenderStackT{texture, nil, nil, true})
}

func (sr *sdlRender) DrawGaugeV(tile types.Tile, topLeft *types.XY, height int32, value, max int, c *types.Colour) {
	sr.drawGaugeV(&sdl.Rect{
		X: (topLeft.X+tile.Left())*sr.glyphSize.X + _PANE_LEFT_MARGIN_OUTER - (sr.glyphSize.X / 2),
		Y: (topLeft.Y+tile.Top())*sr.glyphSize.Y + _PANE_TOP_MARGIN - (sr.glyphSize.Y / 2),
		W: sr.glyphSize.X,
		H: height * sr.glyphSize.Y,
	}, value, max, c)
}

func (sr *sdlRender) drawGaugeV(rect *sdl.Rect, value, max int, c *types.Colour) {
	sr._drawHighlightRect(rect, c, c, 128, 32)

	texture := sr.createRendererTexture()
	if texture == nil {
		return
	}
	rect.H = int32(float64(rect.H) * (float64(value) / float64(max)))
	_ = sr.renderer.SetDrawColor(c.Red, c.Green, c.Blue, 192)
	_ = texture.SetBlendMode(sdl.BLENDMODE_ADD)
	_ = sr.renderer.FillRect(rect)

	sr.renderer.SetRenderTarget(nil)
	sr.AddToOverlayStack(&layer.RenderStackT{texture, nil, nil, true})
}
