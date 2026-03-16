package element_image

import (
	"bytes"
	"errors"
	"fmt"
	"image/png"
	"runtime"

	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/window/backend/cursor"
	"golang.design/x/clipboard"
	"golang.org/x/image/bmp"
)

type elementType int

const (
	_ELEMENT_TYPE_BITMAP = 1 + iota
	_ELEMENT_TYPE_SIXEL
)

type ElementImage struct {
	elType     elementType
	renderer   types.Renderer
	tile       types.Tile
	parameters parametersT
	size       *types.XY
	load       func([]byte, *types.XY) (types.Image, error)
	escSeq     []byte // only used for sixel
	bmp        []byte
	image      types.Image
}

type parametersT struct {
	Base64   string
	Filename string
	Width    int32
	Height   int32
}

func NewBitmap(renderer types.Renderer, tile types.Tile, loadFn func([]byte, *types.XY) (types.Image, error)) *ElementImage {
	return newImage(renderer, tile, loadFn, _ELEMENT_TYPE_BITMAP)
}

func NewSixel(renderer types.Renderer, tile types.Tile, loadFn func([]byte, *types.XY) (types.Image, error)) *ElementImage {
	return newImage(renderer, tile, loadFn, _ELEMENT_TYPE_SIXEL)
}

func newImage(renderer types.Renderer, tile types.Tile, loadFn func([]byte, *types.XY) (types.Image, error), elType elementType) *ElementImage {
	return &ElementImage{
		renderer: renderer,
		tile:     tile,
		load:     loadFn,
		elType:   elType,
	}
}

func (el *ElementImage) Generate(apc *types.ApcSlice) error {
	notify := el.renderer.DisplaySticky(types.NOTIFY_DEBUG, "Importing image from ANSI escape codes....", func() {})
	defer notify.Close()

	apc.Parameters(&el.parameters)

	el.size = new(types.XY)
	el.size.X, el.size.Y = el.parameters.Width, el.parameters.Height

	if el.size.X == 0 && el.size.Y == 0 {
		el.size.Y = 15 // default
	}

	var err error
	switch el.elType {
	case _ELEMENT_TYPE_BITMAP:
		err = el.fromBitmap()
	case _ELEMENT_TYPE_SIXEL:
		err = el.fromSixel(apc)
	default:
		panic("unknown image type")
	}
	if err != nil {
		return fmt.Errorf("cannot decode image: %v", err)
	}

	// cache image

	el.image, err = el.load(el.bmp, el.size)
	if err != nil {
		return fmt.Errorf("cannot cache image: %v", err)
	}

	// destroy image allocation upon garbage collection
	type imageCleanup struct {
		triggerDealloc func(func())
		closeImage     func()
	}
	cleanup := &imageCleanup{
		triggerDealloc: el.renderer.TriggerDeallocation,
		closeImage:     el.image.Close,
	}
	runtime.AddCleanup(el, func(ic *imageCleanup) {
		ic.triggerDealloc(ic.closeImage)
	}, cleanup)
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
func (el *ElementImage) Draw(pos *types.XY) {
	if len(el.bmp) == 0 {
		return
	}

	el.image.Draw(el.tile, el.size, &types.XY{pos.X, pos.Y})
}

func (el *ElementImage) Rune(_ *types.XY) rune {
	return ' '
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
	//callback()
}

func (el *ElementImage) MouseOut() {
	el.renderer.StatusBarText("")
	cursor.Arrow()
}

func (el *ElementImage) MouseHover(_ *types.XY, _ *types.XY) func() {
	return func() {}
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
