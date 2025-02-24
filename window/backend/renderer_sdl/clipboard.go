package rendersdl

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"os"
	"unsafe"

	"github.com/lmorg/mxtty/types"
	"github.com/veandco/go-sdl2/sdl"
	"golang.design/x/clipboard"
)

func (sr *sdlRender) copyRendererToClipboard() {
	defer func() {
		sr.highlighter = nil
		sr.renderer.SetRenderTarget(nil)
		sr.TriggerRedraw()
	}()

	pitch := sr.highlighter.rect.W * 4

	img := image.NewRGBA(image.Rect(0, 0, int(sr.highlighter.rect.W), int(sr.highlighter.rect.H)))

	err := sr.renderer.ReadPixels(sr.highlighter.rect, sdl.PIXELFORMAT_RGBA32, unsafe.Pointer(&img.Pix[0]), int(pitch))
	if err != nil {
		sr.DisplayNotification(types.NOTIFY_ERROR, fmt.Sprintf("Could not copy to clipboard: %s", err.Error()))
		return
	}

	var buf bytes.Buffer

	err = png.Encode(&buf, img)
	if err != nil {
		sr.DisplayNotification(types.NOTIFY_ERROR, fmt.Sprintf("Could not copy to clipboard: %s", err.Error()))
		return
	}

	clipboard.Write(clipboard.FmtImage, buf.Bytes())
	sr.DisplayNotification(types.NOTIFY_INFO, "Copied to clipboard as PNG")
}

func (sr *sdlRender) clipboardPasteText() {
	sr.highlighter = nil
	b := clipboard.Read(clipboard.FmtText)
	if len(b) != 0 {
		sr.term.Reply(b)
		return
	}

	b = clipboard.Read(clipboard.FmtImage)
	if len(b) != 0 {
		f, err := os.CreateTemp("", "*.png")
		if err != nil {
			sr.DisplayNotification(types.NOTIFY_ERROR, err.Error())
			return
		}

		if _, err = f.Write(b); err != nil {
			sr.DisplayNotification(types.NOTIFY_ERROR, err.Error())
			return
		}

		if err = f.Close(); err != nil {
			sr.DisplayNotification(types.NOTIFY_ERROR, err.Error())
			return
		}

		sr.term.Reply([]byte(f.Name()))
		return
	}

	sr.DisplayNotification(types.NOTIFY_WARN, "Clipboard does not contain text to paste")
}
