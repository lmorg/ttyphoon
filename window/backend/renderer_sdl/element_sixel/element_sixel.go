package elementSixel

import (
	"bytes"
	"errors"
	"fmt"
	"image/png"

	"github.com/lmorg/mxtty/types"
	"github.com/lmorg/mxtty/window/backend/cursor"
	"golang.design/x/clipboard"
	"golang.org/x/image/bmp"
)

type ElementImage struct {
	renderer types.Renderer
	tile     types.Tile
	size     *types.XY
	load     func([]byte, *types.XY) (types.Image, error)
	escSeq   []byte
	bmp      []byte
	image    types.Image
}

func New(renderer types.Renderer, tile types.Tile, loadFn func([]byte, *types.XY) (types.Image, error)) *ElementImage {
	return &ElementImage{renderer: renderer, tile: tile, load: loadFn}
}

func (el *ElementImage) Generate(apc *types.ApcSlice) error {
	notify := el.renderer.DisplaySticky(types.NOTIFY_DEBUG, "Importing sixel image from ANSI escape codes....")
	defer notify.Close()

	el.size = new(types.XY)
	if el.size.X == 0 && el.size.Y == 0 {
		el.size.Y = 15 // default
	}

	el.escSeq = []byte(apc.Index(0))
	err := el.decode()
	if err != nil {
		return err
	}

	// cache image

	el.image, err = el.load(el.bmp, el.size)
	if err != nil {
		return fmt.Errorf("cannot cache image: %s", err.Error())
	}
	return nil
}

func (el *ElementImage) Write(_ rune) error {
	return errors.New("not supported")
}

func (el *ElementImage) Size() *types.XY {
	return el.size
}

// Draw:
// size: optional. Defaults to element size
// pos:  required. Position to draw element
func (el *ElementImage) Draw(size *types.XY, pos *types.XY) {
	if len(el.bmp) == 0 {
		return
	}

	if size == nil {
		size = el.size
	}

	el.image.Draw(size, &types.XY{pos.X + el.tile.Left(), pos.Y + el.tile.Top()})
}

func (el *ElementImage) Rune(_ *types.XY) rune {
	return ' '
}

func (el *ElementImage) Close() {
	// clear memory (if required)
	el.image.Close()
}

func (el *ElementImage) MouseClick(_ *types.XY, button types.MouseButtonT, _ uint8, state types.ButtonStateT, callback types.EventIgnoredCallback) {
	if state == types.BUTTON_PRESSED {
		callback()
		return
	}

	switch button {
	case 1:
		err := el.fullscreen()
		if err != nil {
			el.renderer.DisplayNotification(types.NOTIFY_ERROR, "Unable to go fullscreen: "+err.Error())
		}

	case 3:
		el.renderer.AddToContextMenu(types.MenuItem{
			Title: "Copy image to clipboard",
			Fn:    el.copyImageToClipboard,
			Icon:  0xf0c5,
		})
		callback()

	default:
		callback()
	}
}

func (el *ElementImage) MouseWheel(_ *types.XY, _ *types.XY, callback types.EventIgnoredCallback) {
	callback()
}

func (el *ElementImage) MouseMotion(_ *types.XY, _ *types.XY, callback types.EventIgnoredCallback) {
	el.renderer.StatusBarText("[Click] Open image full screen")
	cursor.Hand()
	callback()
}

func (el *ElementImage) MouseOut() {
	el.renderer.StatusBarText("")
	cursor.Arrow()
}

func (el *ElementImage) copyImageToClipboard() {
	bufBmp := bytes.NewBuffer(el.bmp)
	img, err := bmp.Decode(bufBmp)
	if err != nil {
		el.renderer.DisplayNotification(types.NOTIFY_ERROR, fmt.Sprintf("Could not copy to clipboard: %s", err.Error()))
		return
	}

	var bufPng bytes.Buffer
	err = png.Encode(&bufPng, img)
	if err != nil {
		el.renderer.DisplayNotification(types.NOTIFY_ERROR, fmt.Sprintf("Could not copy to clipboard: %s", err.Error()))
		return
	}

	clipboard.Write(clipboard.FmtImage, bufPng.Bytes())
	el.renderer.DisplayNotification(types.NOTIFY_INFO, "Copied to clipboard as PNG")
}
