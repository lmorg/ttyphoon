package elementHyperlink

import (
	"errors"
	"os/exec"

	"github.com/lmorg/mxtty/types"
	"github.com/lmorg/mxtty/window/backend/cursor"
)

type ElementHyperlink struct {
	renderer types.Renderer
	tile     types.Tile
	phrase   []rune
	url      string
	size     *types.XY
	sgr      *types.Sgr
	//fgCol    types.Colour
}

func New(renderer types.Renderer, tile types.Tile) *ElementHyperlink {
	return &ElementHyperlink{renderer: renderer, tile: tile}
}

func (el *ElementHyperlink) Generate(apc *types.ApcSlice, sgr *types.Sgr) error {
	el.url = apc.Index(0)
	if el.url == "" {
		return errors.New("empty url in hyperlink")
	}

	el.phrase = []rune(apc.Index(1))
	if len(el.phrase) == 0 {
		el.phrase = []rune(el.url)
	}

	el.size = &types.XY{int32(len(el.phrase)), 1}
	el.sgr = sgr.Copy()

	return nil
}

func (el *ElementHyperlink) Write(_ rune) error {
	return errors.New("not supported")
}

func (el *ElementHyperlink) Size() *types.XY {
	return el.size
}

// Draw:
// size: optional. Defaults to element size
// pos:  required. Position to draw element
func (el *ElementHyperlink) Draw(size *types.XY, pos *types.XY) {
	for x := range el.size.X {
		cell := &types.Cell{
			Char: el.phrase[x],
			Sgr:  el.sgr,
		}
		el.renderer.PrintCell(el.tile, cell, &types.XY{pos.X + x, pos.Y})
	}
}

func (el *ElementHyperlink) Rune(pos *types.XY) rune {
	return el.phrase[pos.X]
}

func (el *ElementHyperlink) Close() {
	// clear memory (if required)
}

func (el *ElementHyperlink) MouseClick(_ *types.XY, button types.MouseButtonT, _ uint8, state types.ButtonStateT, callback types.EventIgnoredCallback) {
	if button != types.MOUSE_BUTTON_LEFT || state != types.BUTTON_RELEASED {
		callback()
		return
	}

	err := exec.Command("open", el.url).Start()
	if err != nil {
		el.renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
	}
}

func (el *ElementHyperlink) MouseWheel(_ *types.XY, _ *types.XY, callback types.EventIgnoredCallback) {
	callback()
}

func (el *ElementHyperlink) MouseMotion(_ *types.XY, _ *types.XY, callback types.EventIgnoredCallback) {
	el.renderer.StatusBarText("[Click] open " + el.url)
	el.sgr.Bitwise.Set(types.SGR_UNDERLINE)
	cursor.Hand()
}

func (el *ElementHyperlink) MouseOut() {
	el.renderer.StatusBarText("")
	el.sgr.Bitwise.Unset(types.SGR_UNDERLINE)
	cursor.Arrow()
}
