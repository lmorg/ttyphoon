package rendersdl

import (
	"log"
	"sync/atomic"
	"time"

	"github.com/lmorg/mxtty/config"
	"github.com/lmorg/mxtty/types"
	"github.com/lmorg/mxtty/window/backend/renderer_sdl/layer"
	"github.com/veandco/go-sdl2/sdl"
)

func (sr *sdlRender) refreshInterval() {
	if config.Config.Window.RefreshInterval == 0 {
		return
	}

	d := time.Duration(config.Config.Window.RefreshInterval) * time.Millisecond
	for {
		time.Sleep(d)
		sr.TriggerRedraw()
	}
}

func (sr *sdlRender) eventLoop() {
	for {

		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch evt := event.(type) {

			case sdl.WindowEvent:
				sr.eventWindow(&evt)
				sr.TriggerRedraw()

			case sdl.TextInputEvent:
				sr.eventTextInput(&evt)
				sr.TriggerRedraw()

			case sdl.KeyboardEvent:
				sr.eventKeyPress(&evt)
				sr.TriggerRedraw()

			case sdl.MouseButtonEvent:
				sr.eventMouseButton(&evt)
				sr.TriggerRedraw()

			case sdl.MouseMotionEvent:
				sr.eventMouseMotion(&evt)
				// don't trigger redraw

			case sdl.MouseWheelEvent:
				sr.eventMouseWheel(&evt)
				sr.TriggerRedraw()

			case sdl.QuitEvent:
				sr.TriggerQuit()

			}
		}

		select {
		case size := <-sr._resize:
			sr._resizeWindow(size)

		case <-sr._redraw:
			err := render(sr)
			if err != nil {
				log.Printf("ERROR: %s", err.Error())
			}

		case <-sr.pollEventHotkey():
			sr.eventHotkey()

		case <-sr._quit:
			return

		case <-time.After(15 * time.Millisecond):
			continue
		}
	}
}

func (sr *sdlRender) drawBg() {
	if sr.cacheBgTexture != nil {
		return
	}

	/*drawSeparator := func(rect *sdl.Rect, colourBorder, colourFill *types.Colour, alphaBorder, alphaFill byte) {
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
		_ = sr.renderer.FillRect(rect)
	}*/

	sr.cacheBgTexture = sr.createRendererTexture()
	if sr.cacheBgTexture == nil {
		panic("cannot create bg texture")
	}

	w, h := sr.window.GetSize()
	bg := sr.termWin.Active.Term.Bg()
	_ = sr.renderer.SetDrawColor(bg.Red, bg.Green, bg.Blue, 255)
	_ = sr.renderer.FillRect(&sdl.Rect{W: w, H: h})

	for _, tile := range sr.termWin.Tiles {
		if tile.Term == nil {
			continue
		}

		rect := &sdl.Rect{
			X: tile.TopLeft.X*sr.glyphSize.X + _PANE_BLOCK_HIGHLIGHT,
			Y: (tile.TopLeft.Y * sr.glyphSize.Y) + _PANE_TOP_MARGIN - (sr.glyphSize.Y / 2),
			W: (tile.BottomRight.X-tile.TopLeft.X+2)*sr.glyphSize.X + _PANE_LEFT_MARGIN_OUTER - _PANE_BLOCK_HIGHLIGHT,
			H: (tile.BottomRight.Y+2-tile.TopLeft.Y)*sr.glyphSize.Y + _PANE_TOP_MARGIN}

		bg := tile.Term.Bg()
		_ = sr.renderer.SetDrawColor(bg.Red, bg.Green, bg.Blue, 255)
		_ = sr.renderer.FillRect(rect)

		/*if tile.BottomRight.Y < sr.winCellSize.Y-1 || debug.Enabled {
			drawSeparator(&sdl.Rect{
				X: (tile.TopLeft.X * sr.glyphSize.X) + _PANE_LEFT_MARGIN_OUTER,
				Y: ((tile.BottomRight.Y + 1) * sr.glyphSize.Y) + _PANE_TOP_MARGIN + (sr.glyphSize.Y / 2),
				W: ((tile.BottomRight.X - tile.TopLeft.X + 2) * sr.glyphSize.X) + _PANE_LEFT_MARGIN_OUTER,
				H: 0,
			}, types.SGR_COLOUR_BLACK, types.SGR_COLOUR_BLACK, 255, 255)

		}

		if tile.BottomRight.X < sr.winCellSize.X-1 || debug.Enabled {
			drawSeparator(&sdl.Rect{
				X: (tile.BottomRight.X+1)*sr.glyphSize.X + _PANE_LEFT_MARGIN + 2,
				Y: ((tile.TopLeft.Y) * sr.glyphSize.Y) + _PANE_TOP_MARGIN - (sr.glyphSize.Y / 2),
				W: 0,
				H: (tile.BottomRight.Y + 2) * sr.glyphSize.Y,
			}, types.SGR_COLOUR_BLACK, types.SGR_COLOUR_BLACK, 255, 255)
		}*/
	}

	err := sr.renderer.SetRenderTarget(nil)
	if err != nil {
		log.Printf("ERROR: %v", err)
	}

	sr.termWin.Tiles[_TILE_ID_WHOLE_WINDOW] = &types.Tile{TopLeft: &types.XY{}, BottomRight: sr.winCellSize}
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

func (sr *sdlRender) renderStack(stack *[]*layer.RenderStackT) {
	var err error
	for _, item := range *stack {
		err = sr.renderer.Copy(item.Texture, item.SrcRect, item.DstRect)
		if err != nil {
			log.Printf("ERROR: %v", err)
		}
		if item.Destroy {
			_ = item.Texture.Destroy()
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

const _TILE_ID_WHOLE_WINDOW = ""

func render(sr *sdlRender) error {
	defer sr.limiter.Unlock()

	if sr.hidden {
		// window hidden
		return nil
	}

	x, y := sr.window.GetSize()
	rect := &sdl.Rect{W: x, H: y}

	sr.drawBg()
	sr.AddToElementStack(&layer.RenderStackT{sr.cacheBgTexture, nil, nil, false})

	for _, tile := range sr.termWin.Tiles {
		if tile.Term == nil {
			continue
		}

		tile.Term.Render()
	}

	if sr.isMouseInsideWindow() {
		// only run this if mouse cursor is inside the window
		mouseX, mouseY, _ := sdl.GetMouseState()
		tile := sr.getTileFromPxOrActive(mouseX, mouseY)
		posNegX := sr.convertPxToCellXYNegXTile(tile, mouseX, mouseY)
		tile.Term.MousePosition(posNegX)
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

	if sr.highlighter != nil && sr.highlighter.button == 0 {
		sr.copyRendererToClipboard()
		return nil
	}

	switch {
	case sr.inputBox != nil:
		sr.renderInputBox(rect)

	case sr.menu != nil:
		sr.renderMenu(rect)

	default:
		sr.selectionHighlighter()
	}

	sr.renderStack(&sr._overlayStack)

	sr.renderNotification(rect)

	if atomic.CompareAndSwapInt32(&sr.updateTitle, 1, 0) {
		sr.window.SetTitle(sr.title)
	}

	sr.renderer.Present()

	return nil
}
