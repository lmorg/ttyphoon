package element_codeblock

import (
	"fmt"

	"github.com/lmorg/mxtty/ai"
	"github.com/lmorg/mxtty/ai/agent"
	"github.com/lmorg/mxtty/debug"
	"github.com/lmorg/mxtty/types"
	"github.com/lmorg/mxtty/window/backend/cursor"
	"golang.design/x/clipboard"
)

type ElementCodeBlock struct {
	renderer  types.Renderer
	tile      types.Tile
	codeBlock []rune
	size      *types.XY
	sgr       *types.Sgr
}

func New(renderer types.Renderer, tile types.Tile) *ElementCodeBlock {
	return &ElementCodeBlock{renderer: renderer, tile: tile}
}

func (el *ElementCodeBlock) Generate(apc *types.ApcSlice, sgr *types.Sgr) error {
	el.size = &types.XY{int32(len(el.codeBlock)), 1}
	el.sgr = sgr.Copy()

	debug.Log(el.codeBlock)
	debug.Log(len(el.codeBlock))
	debug.Log(el.size)

	return nil
}

func (el *ElementCodeBlock) Write(r rune) error {
	el.codeBlock = append(el.codeBlock, r)
	return nil
}

func (el *ElementCodeBlock) Size() *types.XY {
	return el.size
}

// Draw:
// size: optional. Defaults to element size
// pos:  required. Position to draw element
func (el *ElementCodeBlock) Draw(size *types.XY, pos *types.XY) {
	for x := range el.size.X {
		cell := &types.Cell{
			Char: el.codeBlock[x],
			Sgr:  el.sgr,
		}
		el.renderer.PrintCell(el.tile, cell, &types.XY{pos.X + x, pos.Y})
	}
}

func (el *ElementCodeBlock) Rune(pos *types.XY) rune {
	return el.codeBlock[pos.X]
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
	term := el.tile.GetTerm()
	curPos := term.GetCursorPosition().Y - 1
	meta := agent.Get(el.tile.Id())
	meta.Renderer = el.renderer
	meta.Term = term
	meta.OutputBlock = ""
	meta.InsertAfterRowId = term.GetRowId(curPos)
	meta.CmdLine = string(el.codeBlock)
	return []types.MenuItem{
		{
			Title: "Write code to shell",
			Fn:    func() { term.Reply([]byte(string(el.codeBlock))) },
			Icon:  0xf120,
		},
		{
			Title: "Copy code to clipboard",
			Fn:    func() { copyToClipboard(el.renderer, string(el.codeBlock)) },
			Icon:  0xf0c5,
		},
		{
			Title: fmt.Sprintf("Learn more about code (%s)", meta.ServiceName()),
			Fn:    func() { ai.Explain(meta, true) },
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
	el.sgr.Bitwise.Set(types.SGR_UNDERLINE)
	//el.renderer.DrawHighlightRect(el.tile,)
	cursor.Hand()
}

func (el *ElementCodeBlock) MouseOut() {
	el.renderer.StatusBarText("")
	el.sgr.Bitwise.Unset(types.SGR_UNDERLINE)
	cursor.Arrow()
}
