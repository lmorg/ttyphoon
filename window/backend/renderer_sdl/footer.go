package rendersdl

import (
	"fmt"

	"github.com/lmorg/mxtty/app"
	"github.com/lmorg/mxtty/config"
	"github.com/lmorg/mxtty/types"
	"github.com/lmorg/mxtty/window/backend/cursor"
	"github.com/veandco/go-sdl2/sdl"
)

func (sr *sdlRender) StatusBarText(s string) {
	sr.footerText = s
	sr.TriggerRedraw()
}

func (sr *sdlRender) renderFooter() {
	if sr.footer == 0 {
		return
	}

	_ = sr.createRendererTexture()

	rect := &sdl.Rect{
		X: 0,
		Y: (sr.term.GetSize().Y * sr.glyphSize.Y) + _PANE_TOP_MARGIN,
		W: (sr.term.GetSize().X * sr.glyphSize.X) + (_PANE_LEFT_MARGIN * 3),
		H: (sr.footer * sr.glyphSize.Y) + (_PANE_TOP_MARGIN * 2),
	}

	fill := types.SGR_COLOUR_BLACK_BRIGHT
	_ = sr.renderer.SetDrawColor(fill.Red, fill.Green, fill.Blue, 255)
	_ = sr.renderer.FillRect(rect)

	sr.restoreRendererTexture()

	pos := &types.XY{Y: sr.term.GetSize().Y}

	if !config.Config.Window.StatusBar {
		goto tmuxIntegration
	}

	if sr.footerText == "" {
		sr.footerText = fmt.Sprintf("%s (version %s)  |  [F3] Search%s", app.Title, app.Version(), sr._footerHotkeyMessage())
	}

	sr._footerRenderStatusBar(pos)
	pos.Y++

tmuxIntegration:
	if sr.tmux == nil {
		// This shouldn't happen, but saves a crash in case of this getting
		// invoked before tmux has finished getting set up
		return
	}

	_ = sr.createRendererTexture()
	rect.Y += sr.glyphSize.Y
	_ = sr.renderer.SetDrawColor(fill.Red, fill.Green, fill.Blue, 255)
	_ = sr.renderer.FillRect(rect)
	sr.restoreRendererTexture()

	if sr.windowTabs == nil {
		sr._footerCacheTmuxWindowTabs(pos)
	}

	sr._footerRenderTmuxWindowTabs(pos)
}

func (sr *sdlRender) _footerHotkeyMessage() string {
	if sr.hk == nil {
		return ""
	}

	return "  |  [F12] Show / hide window"
}

func (sr *sdlRender) _footerRenderStatusBar(pos *types.XY) {
	footer := make([]*types.Cell, sr.term.GetSize().X)
	for i := range footer {
		footer[i] = new(types.Cell)
	}

	var i int
	text := []rune(sr.footerText)
	for ; i < len(text) && i < len(footer); i++ {
		footer[i].Char = text[i]
		footer[i].Sgr = types.SGR_DEFAULT.Copy()
	}

	sr.PrintRow(footer[:i], pos)
}

func tabListNewCell(r rune) *types.Cell {
	return &types.Cell{
		Char: r,
		Sgr:  types.SGR_DEFAULT.Copy(),
	}
}

func (sr *sdlRender) _footerCacheTmuxWindowTabs(pos *types.XY) {
	tabList := &tabListT{
		mouseOver: -1,
	}

	heading := []rune("Window tab list → ")

	for _, r := range heading {
		tabList.cells = append(tabList.cells, tabListNewCell(r))
	}

	tabList.boundaries = []int32{0}
	var x int32

	tabList.windows = sr.tmux.RenderWindows()
	for i, win := range tabList.windows {
		if win.Active {
			tabList.active = i
		}

		tabList.cells = append(tabList.cells, tabListNewCell(' '))
		for _, r := range win.Name {
			tabList.cells = append(tabList.cells, tabListNewCell(r))
			x++
		}
		tabList.cells = append(tabList.cells, tabListNewCell(' '))

		x += 2
		tabList.boundaries = append(tabList.boundaries, x)
	}

	tabList.offset = &types.XY{X: int32(len(heading)), Y: pos.Y}

	sr.windowTabs = tabList
}

func (sr *sdlRender) _footerRenderTmuxWindowTabs(pos *types.XY) {
	sr.PrintRow(sr.windowTabs.cells, pos)
	sr.DrawTable(sr.windowTabs.offset, 0, sr.windowTabs.boundaries[1:])

	var (
		topLeftCellX     = sr.windowTabs.offset.X + sr.windowTabs.boundaries[sr.windowTabs.active]
		topLeftCellY     = sr.windowTabs.offset.Y
		bottomRightCellX = sr.windowTabs.boundaries[sr.windowTabs.active+1] - sr.windowTabs.boundaries[sr.windowTabs.active]
		bottomRightCellY = int32(1)
	)

	activeRect := &sdl.Rect{
		X: (topLeftCellX * sr.glyphSize.X) + _PANE_LEFT_MARGIN - 1,
		Y: (topLeftCellY * sr.glyphSize.Y) + _PANE_TOP_MARGIN - 1,
		W: (bottomRightCellX * sr.glyphSize.X) + 2,
		H: (bottomRightCellY * sr.glyphSize.Y) + 2,
	}
	sr._drawHighlightRect(activeRect, highlightBorder, highlightFill, 0, 230)

	if sr.windowTabs.mouseOver == -1 {
		if sr.windowTabs.mouseOver != sr.windowTabs.last {
			sr.windowTabs.last = sr.windowTabs.mouseOver
			cursor.Arrow()
		}
		return
	}

	if sr.windowTabs.mouseOver != sr.windowTabs.last {
		sr.windowTabs.last = sr.windowTabs.mouseOver
		cursor.Hand()
	}

	topLeftCellX = sr.windowTabs.offset.X + sr.windowTabs.boundaries[sr.windowTabs.mouseOver]
	bottomRightCellX = sr.windowTabs.boundaries[sr.windowTabs.mouseOver+1] - sr.windowTabs.boundaries[sr.windowTabs.mouseOver]

	highlightRect := &sdl.Rect{
		X: (topLeftCellX * sr.glyphSize.X) + _PANE_LEFT_MARGIN,
		Y: (topLeftCellY * sr.glyphSize.Y) + _PANE_TOP_MARGIN,
		W: (bottomRightCellX * sr.glyphSize.X),
		H: (bottomRightCellY * sr.glyphSize.Y),
	}
	sr._drawHighlightRect(highlightRect, highlightBorder, highlightFill, highlightAlphaBorder, highlightAlphaFill)
}
