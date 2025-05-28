package element_codeblock

import (
	"bytes"
	"os/exec"

	"github.com/lmorg/mxtty/config"
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
		copyToClipboard(el.renderer, string(el.codeBlock))
		return

	case types.MOUSE_BUTTON_RIGHT:
		el.renderer.AddToContextMenu([]types.MenuItem{
			{
				Title: types.MENU_SEPARATOR,
			},
			{
				Title: "Copy link to clipboard",
				Fn:    func() { copyToClipboard(el.renderer, string(el.codeBlock)) },
				Icon:  0xf0c5,
			},
		}...)
		apps, cmds := config.Config.Terminal.Widgets.AutoHotlink.OpenAgents.MenuItems()
		for i := range apps {
			el.renderer.AddToContextMenu(
				types.MenuItem{
					Title: "Open link with " + apps[i],
					Fn:    func() { openWith(el.renderer, cmds[i], string(el.codeBlock)) },
					Icon:  0xf08e,
				},
			)
		}
		callback()
		return

	default:
		callback()
		return
	}
}

func copyToClipboard(renderer types.Renderer, url string) {
	renderer.DisplayNotification(types.NOTIFY_INFO, "Link copied to clipboard")
	clipboard.Write(clipboard.FmtText, []byte(url))
}

func openWith(renderer types.Renderer, exe []string, url string) {
	var b []byte
	buf := bytes.NewBuffer(b)

	for param := range exe {
		if exe[param] == "$$" {
			exe[param] = url
		}
	}

	cmd := exec.Command(exe[0], exe[1:]...)
	cmd.Stderr = buf

	err := cmd.Start()
	if err != nil {
		renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
		return
	}

	go func() {
		if err := cmd.Wait(); err != nil {
			msg := buf.String()
			if msg == "" {
				msg = err.Error()
			}
			//if debug.Enabled {
			renderer.DisplayNotification(types.NOTIFY_ERROR, msg)
			//}
			//el.renderer.DisplayNotification(types.NOTIFY_INFO, fmt.Sprintf("Unable to launch `%s`", cmds[i-2][0]))
		}
	}()
}

func (el *ElementCodeBlock) MouseWheel(_ *types.XY, _ *types.XY, callback types.EventIgnoredCallback) {
	callback()
}

func (el *ElementCodeBlock) MouseMotion(_ *types.XY, _ *types.XY, callback types.EventIgnoredCallback) {
	el.renderer.StatusBarText("[Click] open " + string(el.codeBlock))
	el.sgr.Bitwise.Set(types.SGR_UNDERLINE)
	//el.renderer.DrawHighlightRect(el.tile,)
	cursor.Hand()
}

func (el *ElementCodeBlock) MouseOut() {
	el.renderer.StatusBarText("")
	el.sgr.Bitwise.Unset(types.SGR_UNDERLINE)
	cursor.Arrow()
}
