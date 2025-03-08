package elementSixel

import (
	"fmt"

	"github.com/lmorg/mxtty/app"
	"github.com/veandco/go-sdl2/sdl"
)

func (el *ElementImage) fullscreen() error {
	mouseX, mouseY, _ := sdl.GetGlobalMouseState()
	window, err := sdl.CreateWindow(
		app.Name,
		mouseX, mouseY, 0, 0,
		sdl.WINDOW_SHOWN|sdl.WINDOW_FULLSCREEN_DESKTOP|sdl.WINDOW_ALWAYS_ON_TOP,
	)
	if err != nil {
		return err
	}
	defer el.renderer.ShowAndFocusWindow()
	defer window.Destroy()

	winID, _ := window.GetID()

	renderer, err := sdl.CreateRenderer(window, -1, sdl.RENDERER_ACCELERATED)
	if err != nil {
		return err
	}
	defer renderer.Destroy()

	imgSurface, ok := el.image.Asset().(*sdl.Surface)
	if !ok {
		return fmt.Errorf("image asset is not a surface")
	}
	//defer imgSurface.Free() // don't free because it's a pointer

	texture, err := renderer.CreateTextureFromSurface(imgSurface)
	if err != nil {
		return err
	}

	winW, winH := window.GetSize()
	srcRect := &sdl.Rect{W: imgSurface.W, H: imgSurface.H}

	imgH := winH
	imgW := int32((float64(imgSurface.W) / float64(imgSurface.H)) * float64(winH))

	if imgW > winW {
		imgW = winW
		imgH = int32((float64(imgSurface.H) / float64(imgSurface.W)) * float64(winW))
	}

	x := (winW / 2) - (imgW / 2)
	y := (winH / 2) - (imgH / 2)
	destRect := &sdl.Rect{X: x, Y: y, W: imgW, H: imgH}

	err = renderer.Copy(texture, srcRect, destRect)
	if err != nil {
		return err
	}

	renderer.Present()

	for {
		for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
			switch evt := event.(type) {

			case sdl.QuitEvent:
				return nil

			case sdl.KeyboardEvent:
				return nil

			case sdl.MouseButtonEvent:
				if evt.State == sdl.PRESSED {
					continue
				}
				return nil

			case sdl.WindowEvent:
				if evt.Event == sdl.WINDOWEVENT_FOCUS_LOST && evt.WindowID == winID {
					return nil
				}
			}
		}

		sdl.Delay(15)
	}
}
