package element_hyperlink

import (
	"errors"
	"fmt"
	"strings"

	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/types"
	menuhyperlink "github.com/lmorg/ttyphoon/utils/menu_hyperlink"
	"github.com/lmorg/ttyphoon/window/backend/cursor"
)

type ElementHyperlink struct {
	renderer types.Renderer
	tile     types.Tile
	label    []rune
	url      string
	scheme   string
	path     string
	sgr      *types.Sgr
	pos      *types.XY
}

func New(renderer types.Renderer, tile types.Tile) *ElementHyperlink {
	return &ElementHyperlink{
		renderer: renderer,
		tile:     tile,
		sgr:      tile.GetTerm().GetSgr().Copy(),
	}
}

func (el *ElementHyperlink) Url() string              { return el.url }
func (el *ElementHyperlink) Scheme() string           { return el.scheme }
func (el *ElementHyperlink) Path() string             { return el.path }
func (el *ElementHyperlink) Label() string            { return string(el.label) }
func (el *ElementHyperlink) Renderer() types.Renderer { return el.renderer }

func (el *ElementHyperlink) Generate(apc *types.ApcSlice) error {
	el.label = []rune(apc.Index(0))
	el.url = apc.Index(1)
	if el.url == "" {
		return errors.New("empty url in hyperlink")
	}

	split := strings.SplitN(el.url, "://", 2)
	if len(split) != 2 {
		return fmt.Errorf("invalid url, missing schema: '://': %s", el.url)
	}
	el.scheme, el.path = strings.ToLower(split[0]), split[1]

	return nil
}

func (el *ElementHyperlink) Write(r rune) error {
	if r == '\n' {
		return nil
	}

	el.label = append(el.label, r)

	return nil
}

func (el *ElementHyperlink) Size() *types.XY {
	panic("not yet implemented")
	//return el.size
}

// Draw:
// pos: Position to draw element
func (el *ElementHyperlink) Draw(termPos *types.XY) {
	el.pos = termPos
	el.sgr = el.tile.GetTerm().GetCellSgr(el.pos)

	width := el.tile.GetTerm().GetSize().X
	x, y := el.pos.X, int32(0)

	for i := range el.label {
		if x >= width {
			y++
			x = 0
		}
		cell := &types.Cell{Sgr: el.sgr}
		cell.Char = el.label[i]
		el.renderer.PrintCell(el.tile, cell, &types.XY{X: x, Y: el.pos.Y + y})
		x++
	}
}

func (el *ElementHyperlink) Rune(pos *types.XY) rune {
	return el.label[pos.X]
}

func (el *ElementHyperlink) MouseClick(_ *types.XY, button types.MouseButtonT, count uint8, state types.ButtonStateT, callback types.EventIgnoredCallback) {
	if state != types.BUTTON_RELEASED {
		callback()
		return
	}

	switch count {
	case 1:
		switch button {
		case types.MOUSE_BUTTON_LEFT:
			menu := el.renderer.NewContextMenu()
			menu.Append(menuhyperlink.MenuItems(el)...)
			menu.DisplayMenu("Hyperlink action")

		case types.MOUSE_BUTTON_RIGHT:
			el.renderer.AddToContextMenu(append([]types.MenuItem{{Title: types.MENU_SEPARATOR}}, menuhyperlink.MenuItems(el)...)...)
			callback()

		case types.MOUSE_BUTTON_MIDDLE:
			el.openWithDefault()
		default:
			callback()
		}

	default:
		switch button {
		case types.MOUSE_BUTTON_LEFT:
			el.openWithDefault()
		default:
			callback()
		}
	}
}

func (el *ElementHyperlink) openWithDefault() {
	_, cmd := config.Config.Terminal.Widgets.AutoHyperlink.OpenAgents.MenuItems(el.scheme)
	if len(cmd) > 0 {
		menuhyperlink.OpenWith(el, cmd[0])
	}
}

func (el *ElementHyperlink) MouseWheel(_ *types.XY, _ *types.XY, callback types.EventIgnoredCallback) {
	callback()
}

func (el *ElementHyperlink) MouseMotion(pos *types.XY, size *types.XY, callback types.EventIgnoredCallback) {
	el.renderer.StatusBarText("[Click] Hyperlink options: " + el.url)
	cursor.Hand()

	if !config.Config.Window.HoverEffectHighlight {
		el.sgr.Bitwise.Set(types.SGR_UNDERLINE)
	}

	/*if strings.HasPrefix(el.scheme, "http") {
		el.renderer.ShowPreview(el.url)
	}*/
}

func (el *ElementHyperlink) MouseOut() {
	//el.renderer.HidePreview()
	el.renderer.StatusBarText("")
	cursor.Arrow()

	if !config.Config.Window.HoverEffectHighlight {
		el.sgr.Bitwise.Unset(types.SGR_UNDERLINE)
	}
}

func (el *ElementHyperlink) MouseHover(_ *types.XY, _ *types.XY) func() {
	if el == nil || !config.Config.Window.HoverEffectHighlight {
		return func() {}
	}

	fn := make([]func(), 0)
	width := el.tile.GetTerm().GetSize().X
	start, x, y := el.pos.X, el.pos.X, int32(0)

	for range el.label {
		if x >= width {
			localStart, localX, localY := start, x, y
			fn = append(fn, func() {
				el.renderer.DrawHighlightRect(
					el.tile,
					&types.XY{X: localStart, Y: el.pos.Y + localY},
					&types.XY{X: localX, Y: 1},
				)
			})
			y++
			x = 0
			start = 0
		}
		x++
	}
	if x > 0 {
		localStart, localX, localY := start, x, y
		fn = append(fn, func() {
			el.renderer.DrawHighlightRect(
				el.tile,
				&types.XY{X: localStart, Y: el.pos.Y + localY},
				&types.XY{X: localX - localStart, Y: 1},
			)
		})
	}

	return func() {
		if el.pos == nil {
			return
		}
		for i := range fn {
			fn[i]()
		}
	}
}
