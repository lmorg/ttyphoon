package element_hyperlink

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/lmorg/mxtty/config"
	"github.com/lmorg/mxtty/types"
	"github.com/lmorg/mxtty/window/backend/cursor"
	"golang.design/x/clipboard"
)

type ElementHyperlink struct {
	renderer types.Renderer
	tile     types.Tile
	phrase   []rune
	url      string
	scheme   string
	path     string
	size     *types.XY
	pos      *types.XY
	sgr      *types.Sgr
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

	split := strings.SplitN(el.url, "://", 2)
	if len(split) != 2 {
		return fmt.Errorf("invalid url, missing '://': %s", el.url)
	}
	el.scheme, el.path = split[0], split[1]

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
func (el *ElementHyperlink) Draw(pos *types.XY) {
	el.pos = pos
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

func (el *ElementHyperlink) MouseClick(_ *types.XY, button types.MouseButtonT, _ uint8, state types.ButtonStateT, callback types.EventIgnoredCallback) {
	if state != types.BUTTON_RELEASED {
		callback()
		return
	}

	switch button {
	case types.MOUSE_BUTTON_LEFT:
		menu := el.renderer.NewContextMenu()
		menu.Append(el.contextMenuItems()...)
		menu.DisplayMenu("Hyperlink action")
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

func (el *ElementHyperlink) contextMenuItems() []types.MenuItem {
	menuItems := []types.MenuItem{
		{
			Title: "Copy link to clipboard",
			Fn:    func() { copyToClipboard(el.renderer, el.url) },
			Icon:  0xf0c5,
		},
	}
	apps, cmds := config.Config.Terminal.Widgets.AutoHotlink.OpenAgents.MenuItems(el.scheme)
	for i := range apps {
		menuItems = append(menuItems,
			types.MenuItem{
				Title: "Open link with " + apps[i],
				Fn:    func() { el.openWith(cmds[i]) },
				Icon:  0xf08e,
			},
		)
	}
	return menuItems
}

func copyToClipboard(renderer types.Renderer, url string) {
	renderer.DisplayNotification(types.NOTIFY_INFO, "Link copied to clipboard")
	clipboard.Write(clipboard.FmtText, []byte(url))
}

func (el *ElementHyperlink) openWith(exe []string) {
	var b []byte
	buf := bytes.NewBuffer(b)

	for param := range exe {
		exe[param] = os.Expand(exe[param], el.getVar)
	}

	cmd := exec.Command(exe[0], exe[1:]...)
	cmd.Stderr = buf

	err := cmd.Start()
	if err != nil {
		el.renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
		return
	}

	go func() {
		if err := cmd.Wait(); err != nil {
			msg := buf.String()
			if msg == "" {
				msg = err.Error()
			}
			//if debug.Enabled {
			el.renderer.DisplayNotification(types.NOTIFY_ERROR, msg)
			//}
			//el.renderer.DisplayNotification(types.NOTIFY_INFO, fmt.Sprintf("Unable to launch `%s`", cmds[i-2][0]))
		}
	}()
}

func (el *ElementHyperlink) getVar(s string) string {
	switch s {
	case "url":
		return el.url
	case "scheme":
		return el.scheme
	case "path":
		return el.path
	default:
		return "INVALID_VARIABLE_NAME"
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
}

func (el *ElementHyperlink) MouseOut() {
	el.renderer.StatusBarText("")
	cursor.Arrow()

	if !config.Config.Window.HoverEffectHighlight {
		el.sgr.Bitwise.Unset(types.SGR_UNDERLINE)
	}
}

func (el *ElementHyperlink) MouseHover() func() {
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
