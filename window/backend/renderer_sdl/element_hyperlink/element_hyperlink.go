package element_hyperlink

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/lmorg/mxtty/ai"
	"github.com/lmorg/mxtty/ai/agent"
	"github.com/lmorg/mxtty/config"
	"github.com/lmorg/mxtty/types"
	"github.com/lmorg/mxtty/utils/runewidth"
	"github.com/lmorg/mxtty/window/backend/cursor"
	"golang.design/x/clipboard"
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

func (el *ElementHyperlink) Generate(apc *types.ApcSlice) error {
	var i int
	el.label = make([]rune, runewidth.StringWidth(apc.Index(0)))
	for _, r := range apc.Index(0) {
		el.label[i] = r
		w := runewidth.RuneWidth(r)
		if w == 2 {
			el.label[i+1] = ' '
		}
		i += w
	}

	el.url = apc.Index(1)
	if el.url == "" {
		return errors.New("empty url in hyperlink")
	}

	split := strings.SplitN(el.url, "://", 2)
	if len(split) != 2 {
		return fmt.Errorf("invalid url, missing '://': %s", el.url)
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
	if sgr := el.tile.GetTerm().GetCellSgr(el.pos); sgr != nil {
		el.sgr = sgr
	}

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
			Fn:    func() { copyLinkToClipboard(el.renderer, el.schemaOrPath()) },
			Icon:  0xf0c5,
		},
	}
	apps, cmds := config.Config.Terminal.Widgets.AutoHyperlink.OpenAgents.MenuItems(el.scheme)
	for i := range apps {
		menuItems = append(menuItems,
			types.MenuItem{
				Title: "Open link with " + apps[i],
				Fn:    func() { el.openWith(cmds[i]) },
				Icon:  0xf08e,
			},
		)
	}
	menuItems = append(menuItems, []types.MenuItem{
		{
			Title: types.MENU_SEPARATOR,
		},
		{
			Title: "Write link to shell",
			Fn:    func() { el.tile.GetTerm().Reply([]byte(el.schemaOrPath())) },
			Icon:  0xf120,
		},
	}...,
	)

	if strings.HasPrefix(el.scheme, "http") {
		term := el.tile.GetTerm()
		curPos := term.GetCursorPosition().Y - 1
		meta := agent.Get(el.tile.Id())
		meta.Renderer = el.renderer
		meta.Term = term
		meta.OutputBlock = ""
		meta.InsertAfterRowId = term.GetRowId(curPos)
		meta.CmdLine = string(el.url)
		menuItems = append(menuItems, types.MenuItem{
			Title: fmt.Sprintf("Summarize hyperlink (%s)", meta.ServiceName()),
			Fn: func() {
				ai.AskAI(meta, fmt.Sprintf("Can you summarize the contents of this web page: %s\n Do NOT to check other websites nor use any search engines.", el.url))
			},
			Icon: 0xf544,
		})
	}

	if el.scheme == "file" {
		f, err := os.Open(el.schemaOrPath())
		if err != nil {
			el.renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
			return menuItems
		}
		defer f.Close()

		info, err := f.Stat()
		if err != nil {
			el.renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
			return menuItems
		}

		if info.IsDir() || info.Size() > _CONTENTS_CLIP_MAX {
			return menuItems
		}

		menuItems = append(menuItems, types.MenuItem{
			Title: "Copy contents to clipboard",
			Fn:    func() { copyContentsToClipboard(el.renderer, el.schemaOrPath()) },
			Icon:  0xf0c5,
		})
	}

	return menuItems
}

func (el *ElementHyperlink) schemaOrPath() string {
	if el.scheme == "file" {
		return string(el.path)
	} else {
		return string(el.url)
	}
}

func copyLinkToClipboard(renderer types.Renderer, url string) {
	renderer.DisplayNotification(types.NOTIFY_INFO, "Link copied to clipboard")
	clipboard.Write(clipboard.FmtText, []byte(url))
}

const _CONTENTS_CLIP_MAX = 10 * 1024 * 1024 // 10 MB

func copyContentsToClipboard(renderer types.Renderer, path string) {
	f, err := os.Open(path)
	if err != nil {
		renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
		return
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
		return
	}

	if info.Size() > _CONTENTS_CLIP_MAX {
		renderer.DisplayNotification(types.NOTIFY_WARN, "file too large")
		return
	}

	b, err := io.ReadAll(f)
	if err != nil {
		renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
		return
	}

	renderer.DisplayNotification(types.NOTIFY_INFO, "File contents copied to clipboard")
	clipboard.Write(clipboard.FmtText, b)
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
			el.renderer.DisplayNotification(types.NOTIFY_ERROR, msg)
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

func (el *ElementHyperlink) MouseHover(_ *types.XY, _ *types.XY) func() {
	if !config.Config.Window.HoverEffectHighlight {
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
