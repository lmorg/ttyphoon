package rendersdl

import (
	"log"
	godebug "runtime/debug"
	"sync/atomic"

	"github.com/lmorg/mxtty/config"
	"github.com/lmorg/mxtty/types"
	"github.com/lmorg/mxtty/window/backend/renderer_sdl/layer"
	"github.com/veandco/go-sdl2/sdl"
)

type bgT struct{ _texture *sdl.Texture }

func (bg *bgT) IsSet() bool              { return bg._texture != nil }
func (bg *bgT) Set(texture *sdl.Texture) { bg._texture = texture }
func (bg *bgT) Destroy(sr *sdlRender) {
	destroy := func() {
		bg._texture.Destroy()
		bg._texture = nil
		go godebug.FreeOSMemory()
	}
	go func() { sr._deallocStack <- destroy }()
}

func (sr *sdlRender) drawBg() {
	if sr.cacheBgTexture.IsSet() {
		return
	}

	sr.cacheBgTexture.Set(sr.createRendererTexture())
	if !sr.cacheBgTexture.IsSet() {
		panic("cannot create bg texture")
	}

	w, h := sr.window.GetSize()
	canvasBg := types.SGR_COLOR_BACKGROUND
	//if len(sr.termWin.Tiles) <= 1 {
	//	canvasBg = sr.termWin.Active.GetTerm().Bg()
	//}

	_ = sr.renderer.SetDrawColor(canvasBg.Red, canvasBg.Green, canvasBg.Blue, 255)
	_ = sr.renderer.FillRect(&sdl.Rect{W: w, H: h})

	if len(sr.termWin.Tiles) > 1 && !config.Config.Window.TileHighlightFill {
		canvasBg := types.SGR_COLOR_FOREGROUND
		_ = sr.renderer.SetDrawColor(canvasBg.Red, canvasBg.Green, canvasBg.Blue, 16)
		_ = sr.renderer.FillRect(&sdl.Rect{X: _PANE_LEFT_MARGIN_OUTER + sr.glyphSize.X, Y: _PANE_TOP_MARGIN, W: sr.winCellSize.X * sr.glyphSize.X, H: sr.winCellSize.Y * sr.glyphSize.Y})
	}

	for _, tile := range sr.termWin.Tiles {
		rect := &sdl.Rect{
			X: tile.Left()*sr.glyphSize.X + _PANE_BLOCK_HIGHLIGHT + _PANE_LEFT_MARGIN_OUTER,
			Y: (tile.Top() * sr.glyphSize.Y) + _PANE_TOP_MARGIN, // - _PANE_BLOCK_HIGHLIGHT,
			W: (tile.Right()-tile.Left()+2)*sr.glyphSize.X - _PANE_BLOCK_HIGHLIGHT,
			H: (tile.Bottom()+2-tile.Top())*sr.glyphSize.Y - _PANE_BLOCK_HIGHLIGHT}

		//bg := tile.GetTerm().Bg()
		bg := types.SGR_COLOR_BACKGROUND
		_ = sr.renderer.SetDrawColor(bg.Red, bg.Green, bg.Blue, 255)
		_ = sr.renderer.FillRect(rect)
	}

	if len(sr.termWin.Tiles) > 1 {
		tile := sr.termWin.Active
		rect := &sdl.Rect{
			X: tile.Left()*sr.glyphSize.X + _PANE_LEFT_MARGIN_OUTER,
			Y: (tile.Top() * sr.glyphSize.Y) + _PANE_TOP_MARGIN,
			W: (tile.Right() - tile.Left() + 2) * sr.glyphSize.X, //- _PANE_BLOCK_HIGHLIGHT,
			H: (tile.Bottom()+2-tile.Top())*sr.glyphSize.Y - _PANE_BLOCK_HIGHLIGHT}
		if tile.AtBottom() {
			rect.H -= sr.glyphSize.Y/2 + _PANE_BLOCK_HIGHLIGHT
		}

		if config.Config.Window.TileHighlightFill {
			_ = sr.renderer.SetDrawColor(types.COLOR_SELECTION.Red, types.COLOR_SELECTION.Blue, types.COLOR_SELECTION.Blue, 64)
			_ = sr.renderer.FillRect(rect)
		} else {
			_ = sr.renderer.SetDrawColor(types.COLOR_SELECTION.Red, types.COLOR_SELECTION.Blue, types.COLOR_SELECTION.Blue, highlightAlphaBorder)
			_ = sr.renderer.DrawRect(rect)
		}
	}

	err := sr.renderer.SetRenderTarget(nil)
	if err != nil {
		log.Printf("ERROR: %v", err)
	}
}

func (sr *sdlRender) AddToElementStack(item *layer.RenderStackT) {
	sr._elementStack = append(sr._elementStack, item)
}

func (sr *sdlRender) AddToOverlayStack(item *layer.RenderStackT) {
	sr._overlayStack = append(sr._overlayStack, item)
}

func (sr *sdlRender) createRendererTexture() *sdl.Texture {
	w, h, err := sr.renderer.GetOutputSize()
	if err != nil {
		log.Printf("ERROR: %v", err)
		return nil
	}
	texture, err := sr.renderer.CreateTexture(sdl.PIXELFORMAT_RGBA32, sdl.TEXTUREACCESS_TARGET, w, h)
	if err != nil {
		log.Printf("ERROR: %v", err)
		return nil
	}
	err = sr.renderer.SetRenderTarget(texture)
	if err != nil {
		log.Printf("ERROR: %v", err)
		return nil
	}
	err = texture.SetBlendMode(sdl.BLENDMODE_BLEND)
	if err != nil {
		log.Printf("ERROR: %v", err)
		return nil
	}
	return texture
}

func (sr *sdlRender) restoreRendererTexture() {
	texture := sr.renderer.GetRenderTarget()
	sr.AddToElementStack(&layer.RenderStackT{texture, nil, nil, true})
	err := sr.renderer.SetRenderTarget(nil)
	if err != nil {
		log.Printf("ERROR: %v", err)
	}
}

func (sr *sdlRender) restoreRendererTextureCrop(tile types.Tile) {
	if tile.GetTerm() == nil {
		sr.restoreRendererTexture()
		return
	}

	size := tile.GetTerm().GetSize()

	src := &sdl.Rect{
		X: _PANE_LEFT_MARGIN,
		Y: _PANE_TOP_MARGIN,
		W: size.X * sr.glyphSize.X,
		H: size.Y * sr.glyphSize.Y,
	}

	dst := &sdl.Rect{
		X: tile.Left()*sr.glyphSize.X + _PANE_LEFT_MARGIN,
		Y: tile.Top()*sr.glyphSize.Y + _PANE_TOP_MARGIN,
		W: size.X * sr.glyphSize.X,
		H: size.Y * sr.glyphSize.Y,
	}

	texture := sr.renderer.GetRenderTarget()
	sr.AddToElementStack(&layer.RenderStackT{texture, src, dst, true})
	err := sr.renderer.SetRenderTarget(nil)
	if err != nil {
		log.Printf("ERROR: %v", err)
	}
}

func (sr *sdlRender) renderStack(stack *[]*layer.RenderStackT) {
	var err error
	for _, item := range *stack {
		err = sr.renderer.Copy(item.Texture, item.SrcRect, item.DstRect)
		if err != nil {
			log.Printf("ERROR: %v", err)
		}
		if item.Destroy {
			_ = item.Texture.Destroy()
			//sr.TriggerDeallocation(func() { item.Texture.Destroy() })
		}
	}
	*stack = make([]*layer.RenderStackT, 0) // clear image stack
}

func (sr *sdlRender) isMouseInsideWindow() bool {
	x, y := sr.window.GetSize()
	mouseGX, mouseGY, _ := sdl.GetGlobalMouseState()
	winGX, winGY := sr.window.GetPosition()
	return mouseGX >= winGX && mouseGY >= winGY && mouseGX <= winGX+x && mouseGY <= winGY+y
}

func render(sr *sdlRender) error {
	//if sr.termWin == nil || !sr.limiter.TryLock() {
	//	return nil
	//}
	defer sr.limiter.Unlock()
	//defer sr.renderLock.Store(false)

	if sr.hidden {
		// window hidden
		return nil
	}

	x, y := sr.window.GetSize()
	rect := &sdl.Rect{W: x, H: y}

	sr.drawBg()
	sr.AddToElementStack(&layer.RenderStackT{sr.cacheBgTexture._texture, nil, nil, false})

	for _, tile := range sr.termWin.Tiles {
		ok := tile.GetTerm().Render()
		if !ok {
			return nil
		}
	}

	if sr.isMouseInsideWindow() {
		// only run this if mouse cursor is inside the window
		mouseX, mouseY, _ := sdl.GetMouseState()
		tile := sr.getTileFromPxOrActive(mouseX, mouseY)
		posNegX := sr.convertPxToCellXYNegXTile(tile, mouseX, mouseY)
		tile.GetTerm().MousePosition(posNegX)
	}

	sr.renderFooter()

	if sr.highlighter != nil && sr.highlighter.button == 0 {
		texture := sr.createRendererTexture()
		if texture == nil {
			sr.highlighter = nil
			return nil
		}
		defer texture.Destroy()
	}

	sr.renderStack(&sr._elementStack)

	switch {
	case sr.inputBox != nil:
		sr.renderInputBox(rect)

	case sr.menu != nil:
		sr.renderMenu(rect)
	}

	sr.renderStack(&sr._overlayStack)

	if sr.highlighter != nil && sr.highlighter.button == 0 {
		sr.copyRendererToClipboard()
		return nil
	}

	sr.selectionHighlighter()

	sr.renderNotification(rect)

	if atomic.CompareAndSwapInt32(&sr.updateTitle, 1, 0) {
		sr.window.SetTitle(sr.title)
	}

	sr.renderer.Present()

	return nil
}
