package mainwindow

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/utils/jupyter"
	runtime "github.com/lmorg/ttyphoon/utils/wailsruntime"
	renderwebkit "github.com/lmorg/ttyphoon/window/backend/renderer_webkit"
)

func (a *WApp) GetTerminalGlyphSize() *types.XY {
	renderer, ok := renderwebkit.CurrentRenderer()
	if ok {
		glyphSize := renderer.GetGlyphSize()
		if glyphSize != nil {
			return glyphSize
		}
	}

	return nil
}

func (a *WApp) TerminalMouseButton(cellX, cellY int32, button int, clicks int, pressed bool) {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return
	}

	state := types.BUTTON_RELEASED
	if pressed {
		state = types.BUTTON_PRESSED
	}

	renderer.HandleMouseButton(cellX, cellY, types.MouseButtonT(button), uint8(clicks), state)
}

func (a *WApp) TerminalMouseWheel(cellX, cellY, moveX, moveY int32) {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return
	}

	renderer.HandleMouseWheel(cellX, cellY, moveX, moveY)
}

func (a *WApp) TerminalMouseLeave() {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return
	}
	renderer.HandleMouseLeave()
}

func (a *WApp) TerminalMouseMotion(cellX, cellY, relX, relY, state int32) {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return
	}

	renderer.HandleMouseMotion(cellX, cellY, relX, relY, state)
}

func (a *WApp) TerminalSetFocus(focused bool) {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return
	}

	tile := renderer.ActiveTile()
	if tile == nil || tile.GetTerm() == nil {
		return
	}

	tile.GetTerm().SetFocus(focused)
	renderer.TriggerRedraw()
}

func (a *WApp) FocusTerminalPane() {
	runtime.EventsEmit(a.ctx, "focusTerminalPane")
}

func (a *WApp) TerminalRequestRedraw() {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return
	}

	renderer.TriggerRedraw()
}

func (a *WApp) CloseNotification(id int64) {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return
	}

	renderer.CloseNotification(id)
}

func (a *WApp) TerminalMenuHighlight(menuID, index int) {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return
	}

	renderer.MenuHighlight(menuID, index)
}

func (a *WApp) TerminalMenuSelect(menuID, index int) {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return
	}

	renderer.MenuSelect(menuID, index)
}

func (a *WApp) TerminalMenuCancel(menuID, index int) {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return
	}

	renderer.MenuCancel(menuID, index)
}

func (a *WApp) TerminalTextInput(text string) {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return
	}

	renderer.HandleTextInput(text)
}

func (a *WApp) TerminalResize(cols, rows int32) {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return
	}

	renderer.WindowResized(cols, rows)
	renderer.TriggerRedraw()
}

func (a *WApp) TerminalSetGlyphSize(width, height int32) {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return
	}

	renderer.SetGlyphSize(width, height)
}

func (a *WApp) TerminalCopyImageDataURL(dataURL string) error {
	if dataURL == "" {
		return fmt.Errorf("empty image data URL")
	}

	comma := strings.IndexByte(dataURL, ',')
	if comma <= 0 || comma >= len(dataURL)-1 {
		return fmt.Errorf("invalid image data URL")
	}

	meta := dataURL[:comma]
	if !strings.Contains(meta, ";base64") {
		return fmt.Errorf("image data URL is not base64 encoded")
	}

	png, err := base64.StdEncoding.DecodeString(dataURL[comma+1:])
	if err != nil {
		return fmt.Errorf("decode image data URL: %w", err)
	}

	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return fmt.Errorf("renderer unavailable")
	}

	return renderer.CopyImageToClipboard(png)
}

func (a *WApp) TerminalGetTabs() []map[string]any {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return nil
	}

	tabs := renderer.GetWindowTabs()
	out := make([]map[string]any, 0, len(tabs))

	activePaneId := renderer.GetActivePaneId()

	for i := range tabs {
		out = append(out, map[string]any{
			"id":           tabs[i].ID,
			"name":         tabs[i].Name,
			"index":        tabs[i].Index,
			"active":       tabs[i].Active,
			"activePaneId": activePaneId,
		})
	}

	return out
}

func (a *WApp) TerminalGetPanes() []renderwebkit.TerminalPane {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return nil
	}
	return renderer.GetPaneLayout()
}

func (a *WApp) TerminalCancelPaneAutoClose(paneID string) error {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return fmt.Errorf("renderer not available")
	}
	return renderer.CancelPaneAutoClose(paneID)
}

func (a *WApp) TerminalClonePane() error {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return fmt.Errorf("renderer not available")
	}
	return renderer.ClonePaneAtCurrentDirectory()
}

func (a *WApp) TerminalClosePane(paneID string) error {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return fmt.Errorf("renderer not available")
	}
	return renderer.CloseCompletedPane(paneID)
}

func (a *WApp) TerminalConfirmKillPane(paneID, paneName string) error {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return fmt.Errorf("renderer not available")
	}
	return renderer.ConfirmKillPane(paneID, paneName)
}

func (a *WApp) TerminalSelectWindow(windowID string) {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return
	}

	renderer.SelectWindow(windowID)
}

func (a *WApp) TerminalPaneZoom(paneID string) error {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return fmt.Errorf("renderer not available")
	}
	return renderer.ZoomPane(paneID)
}

func (a *WApp) TerminalKeyPress(key string, ctrl, alt, shift, meta bool) {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return
	}

	renderer.HandleKeyPress(key, ctrl, alt, shift, meta)
}

func (a *WApp) TerminalInputBoxSubmit(id int64, v *types.InputBoxCallbackResultT, isOk bool) {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return
	}

	renderer.InputBoxSubmit(id, v, isOk)
}

func (a *WApp) TerminalInputBoxHistoryImages(id int64, index int) ([]types.ImageAttachment, error) {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return nil, errors.New("terminal renderer is unavailable")
	}
	return renderer.InputBoxHistoryImages(id, index)
}

func (a *WApp) startTerminalWindow() {
	err := startBackend(a)
	if err != nil {
		panic(err)
	}

	// Set app reference on renderer for hotkey handlers
	if wr, ok := renderwebkit.CurrentRenderer(); ok {
		wr.SetApp(a)
		go a.prewarmNotesFiles(wr)
	}
}

func (a *WApp) SendToTerminal(content, language string) error {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return fmt.Errorf("renderer not available")
	}

	command, cleanup, err := jupyter.PrepareTerminalCommand(content, language, a.GetCurrentProject())
	if err != nil {
		return err
	}
	err = renderer.ClonePane(command, func(_ []byte, _ int, err error) {
		cleanup()
		if err != nil {
			renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
		}
	})
	if err != nil {
		cleanup()
	}
	return err
}
