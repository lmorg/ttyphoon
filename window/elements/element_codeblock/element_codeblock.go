package element_codeblock

import (
	"fmt"

	"github.com/lmorg/ttyphoon/ai"
	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/window/backend/cursor"
	"golang.design/x/clipboard"
)

type ElementCodeBlock struct {
	renderer types.Renderer
	tile     types.Tile
	raw      []rune
	grid     [][]*types.Cell
	size     *types.XY
	pos      *types.XY
}

func New(renderer types.Renderer, tile types.Tile) *ElementCodeBlock {
	return &ElementCodeBlock{
		renderer: renderer,
		tile:     tile,
		size:     &types.XY{X: tile.GetTerm().GetSize().X, Y: 1},
		grid:     make([][]*types.Cell, 1),
	}
}

func (el *ElementCodeBlock) Generate(apc *types.ApcSlice) error {
	if el.size.Y == 1 {
		el.size = &types.XY{int32(len(el.raw)), 1}
	}

	return nil
}

func (el *ElementCodeBlock) Write(r rune) error {
	cell := &types.Cell{Char: r, Sgr: el.tile.GetTerm().GetSgr().Copy()}
	el.raw = append(el.raw, cell.Char)
	if cell.Char == '\n' {
		el.size.Y++
		el.grid = append(el.grid, []*types.Cell{})
	} else {
		el.grid[len(el.grid)-1] = append(el.grid[len(el.grid)-1], cell)
	}

	return nil
}

func (el *ElementCodeBlock) Size() *types.XY {
	return el.size
}

// Draw:
// pos:  required. Position to draw element
func (el *ElementCodeBlock) Draw(pos *types.XY) {
	el.pos = pos
	for y := range el.grid {
		for x := range el.grid[y] {
			el.renderer.PrintCell(el.tile, el.grid[y][x], &types.XY{pos.X + int32(x), pos.Y + int32(y)})
		}
	}
}

func (el *ElementCodeBlock) Rune(pos *types.XY) rune {
	line := el.grid[pos.Y]
	if len(line) <= int(pos.X) {
		return ' '
	}
	return line[pos.X].Char
}

func (el *ElementCodeBlock) MouseClick(_ *types.XY, button types.MouseButtonT, _ uint8, state types.ButtonStateT, callback types.EventIgnoredCallback) {
	if state != types.BUTTON_RELEASED {
		callback()
		return
	}

	switch button {
	case types.MOUSE_BUTTON_LEFT:
		menu := el.renderer.NewContextMenu()
		menu.Append(el.contextMenuItems()...)
		menu.DisplayMenu("Code block action")
		return

	case types.MOUSE_BUTTON_RIGHT:
		el.renderer.AddToContextMenu(append([]types.MenuItem{{Title: types.MENU_SEPARATOR}}, el.contextMenuItems()...)...)
		callback()
		return

	default:
		callback()
		return
	}
}

func (el *ElementCodeBlock) contextMenuItems() []types.MenuItem {
	agt := agent.Get(el.tile.Id())
	agt.Meta = &agent.Meta{
		CmdLine: string(el.raw),
	}

	return []types.MenuItem{
		{
			Title: "Write code to shell",
			Fn:    func() { el.tile.GetTerm().Reply([]byte(string(el.raw))) },
			Icon:  0xf120,
		},
		{
			Title: "Copy code to clipboard",
			Fn:    func() { copyToClipboard(el.renderer, string(el.raw)) },
			Icon:  0xf0c5,
		},
		{
			Title: fmt.Sprintf("Learn more about code (%s)", agt.ServiceName()),
			Fn:    func() { ai.Explain(agt, true) },
			Icon:  0xf544,
		},
	}
}

func copyToClipboard(renderer types.Renderer, code string) {
	renderer.DisplayNotification(types.NOTIFY_INFO, "Code copied to clipboard")
	clipboard.Write(clipboard.FmtText, []byte(code))
}

func (el *ElementCodeBlock) MouseWheel(_ *types.XY, _ *types.XY, callback types.EventIgnoredCallback) {
	callback()
}

func (el *ElementCodeBlock) MouseMotion(_ *types.XY, _ *types.XY, callback types.EventIgnoredCallback) {
	el.renderer.StatusBarText("[Click] Code block options...")
	cursor.Hand()

	if !config.Config.Window.HoverEffectHighlight {
		el.bitwise(func(bitwise *types.SgrFlag) { bitwise.Set(types.SGR_UNDERLINE) })
	}
}

func (el *ElementCodeBlock) MouseOut() {
	el.renderer.StatusBarText("")
	cursor.Arrow()

	if !config.Config.Window.HoverEffectHighlight {
		//el.sgr.Bitwise.Unset(types.SGR_UNDERLINE)
		el.bitwise(func(bitwise *types.SgrFlag) { bitwise.Unset(types.SGR_UNDERLINE) })
	}
}

func (el *ElementCodeBlock) MouseHover(_ *types.XY, _ *types.XY) func() {
	if !config.Config.Window.HoverEffectHighlight {
		return func() {}
	}

	return func() {
		if el.pos == nil {
			return
		}
		el.renderer.DrawHighlightRect(el.tile, el.pos, el.size)
	}
}

func (el *ElementCodeBlock) bitwise(fn func(*types.SgrFlag)) {
	for y := range el.grid {
		for x := range el.grid[y] {
			fn(&(el.grid[y][x].Sgr.Bitwise))
		}
	}
}
