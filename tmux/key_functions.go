package tmux

import (
	"fmt"
	"strings"

	"github.com/lmorg/ttyphoon/codes"
	"github.com/lmorg/ttyphoon/hotkeys"
	"github.com/lmorg/ttyphoon/types"
)

type terminalPaneTabsProvider interface {
	TerminalPaneTabs() []types.TerminalPaneTab
	ActivateTerminalPaneTab(string)
}

func fnKeyNewWindow(tmux *Tmux) error {
	_, err := tmux.SendCommand([]byte("new-window"))
	return err
}

func fnKeyKillPane(tmux *Tmux) error {
	command := fmt.Sprintf("kill-pane -t %s", tmux.ActivePane().id)
	_, err := tmux.SendCommand([]byte(command))
	return err
}

func fnKeyKillCurrentWindow(tmux *Tmux) error {
	command := fmt.Sprintf("kill-window -t %s", tmux.activeWindow.id)
	_, err := tmux.SendCommand([]byte(command))
	return err
}

func fnKeyRenameWindow(tmux *Tmux) error {
	tmux.renderer.DisplayInputBox("Please enter a new name for this window", tmux.activeWindow.name, func(name string) {
		err := tmux.RenameWindow(tmux.activeWindow, name)
		if err != nil {
			tmux.renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
		}
	}, nil)
	return nil
}

func fnKeyChooseWindowFromList(tmux *Tmux) error {
	type menuEntry struct {
		name string
		id   string
		aux  bool
	}

	entries := make([]menuEntry, 0, len(tmux.appWindow.Tabs)+1)
	for i := range tmux.appWindow.Tabs {
		entries = append(entries, menuEntry{
			name: tmux.appWindow.Tabs[i].Name(),
			id:   tmux.appWindow.Tabs[i].Id(),
		})
	}

	if provider, ok := tmux.renderer.(terminalPaneTabsProvider); ok {
		extraTabs := provider.TerminalPaneTabs()
		for i := range extraTabs {
			if extraTabs[i].ID == "" {
				continue
			}
			entries = append(entries, menuEntry{
				name: extraTabs[i].Name,
				id:   extraTabs[i].ID,
				aux:  true,
			})
		}
	}

	if len(entries) == 0 {
		return nil
	}

	windowNames := make([]string, len(entries))
	for i := range entries {
		windowNames[i] = entries[i].name
	}

	activeWindow := tmux.activeWindow.id
	previewWindowID := ""

	restorePreviewCursor := func() {
		if previewWindowID == "" {
			return
		}

		win := tmux.wins.Get(previewWindowID)
		if win != nil && win.activePane != nil && win.activePane.term != nil {
			win.activePane.term.ShowCursor(true)
		}

		previewWindowID = ""
	}

	_highlightCallback := func(i int) {
		if i < 0 || i >= len(entries) {
			return
		}

		entry := entries[i]
		if entry.aux {
			restorePreviewCursor()
			if provider, ok := tmux.renderer.(terminalPaneTabsProvider); ok {
				provider.ActivateTerminalPaneTab(entry.id)
			}
			return
		}

		if provider, ok := tmux.renderer.(terminalPaneTabsProvider); ok {
			provider.ActivateTerminalPaneTab("__tmux__")
		}

		targetWindowID := entry.id
		if tmux.activeWindow.id == targetWindowID {
			return
		}

		// Ensure the previously previewed window cursor is restored before
		// previewing another one.
		restorePreviewCursor()

		err := tmux.SelectAndResizeWindow(targetWindowID, tmux.renderer.GetWindowSizeCells())
		if err != nil {
			tmux.renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
		}

		win := tmux.wins.Get(targetWindowID)
		if win != nil && win.activePane != nil && win.activePane.term != nil {
			win.activePane.term.ShowCursor(false)
			previewWindowID = targetWindowID
		}
	}

	_chooseCallback := func(i int) {
		if i < 0 || i >= len(entries) {
			return
		}

		restorePreviewCursor()

		entry := entries[i]
		if entry.aux {
			if provider, ok := tmux.renderer.(terminalPaneTabsProvider); ok {
				provider.ActivateTerminalPaneTab(entry.id)
			}
			return
		}

		if provider, ok := tmux.renderer.(terminalPaneTabsProvider); ok {
			provider.ActivateTerminalPaneTab("__tmux__")
		}

		targetWindowID := entry.id
		err := tmux.SelectAndResizeWindow(targetWindowID, tmux.renderer.GetWindowSizeCells())
		if err != nil {
			tmux.renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
		}

		win := tmux.wins.Get(targetWindowID)
		if win != nil && win.activePane != nil && win.activePane.term != nil {
			win.activePane.term.ShowCursor(true)
		}
		//tmux.renderer.UpdateNotes(tmux.activeWindow.ActivePane())
	}

	_cancelCallback := func(_ int) {
		restorePreviewCursor()

		if provider, ok := tmux.renderer.(terminalPaneTabsProvider); ok {
			provider.ActivateTerminalPaneTab("__tmux__")
		}

		err := tmux.selectWindow(activeWindow)
		if err != nil {
			tmux.renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
		}

		win := tmux.wins.Get(activeWindow)
		if win != nil && win.activePane != nil && win.activePane.term != nil {
			win.activePane.term.ShowCursor(true)
		}
	}

	tmux.renderer.DisplayMenu("Choose a window", windowNames, _highlightCallback, _chooseCallback, _cancelCallback)
	return nil
}

func fnKeySelectWindow0(tmux *Tmux) error { return _fnKeySelectWindow(tmux, 0) }
func fnKeySelectWindow1(tmux *Tmux) error { return _fnKeySelectWindow(tmux, 1) }
func fnKeySelectWindow2(tmux *Tmux) error { return _fnKeySelectWindow(tmux, 2) }
func fnKeySelectWindow3(tmux *Tmux) error { return _fnKeySelectWindow(tmux, 3) }
func fnKeySelectWindow4(tmux *Tmux) error { return _fnKeySelectWindow(tmux, 4) }
func fnKeySelectWindow5(tmux *Tmux) error { return _fnKeySelectWindow(tmux, 5) }
func fnKeySelectWindow6(tmux *Tmux) error { return _fnKeySelectWindow(tmux, 6) }
func fnKeySelectWindow7(tmux *Tmux) error { return _fnKeySelectWindow(tmux, 7) }
func fnKeySelectWindow8(tmux *Tmux) error { return _fnKeySelectWindow(tmux, 8) }
func fnKeySelectWindow9(tmux *Tmux) error { return _fnKeySelectWindow(tmux, 9) }
func _fnKeySelectWindow(tmux *Tmux, i int) error {
	if i >= len(tmux.appWindow.Tabs) {
		return fmt.Errorf("there is not a window %d", i)
	}

	return tmux.selectWindow(tmux.appWindow.Tabs[i].Id())
}

func fnKeyLastPane(tmux *Tmux) error {
	_, err := tmux.SendCommand([]byte("last-pane"))
	return err
}

func fnKeyLastWindow(tmux *Tmux) error {
	_, err := tmux.SendCommand([]byte("last-window"))
	return err
}

func fnKeyNextWindowAlert(tmux *Tmux) error {
	_, err := tmux.SendCommand([]byte("next-window -a"))
	return err
}

func fnKeyPreviousWindowAlert(tmux *Tmux) error {
	_, err := tmux.SendCommand([]byte("previous-window -a"))
	return err
}

func _fnKeySplitWindow(tmux *Tmux, flag string) error {
	_, err := tmux.SendCommand([]byte("split-window " + flag))
	//go tmux.renderer.RefreshWindowList()
	return err
}
func fnKeySplitWindowHorizontally(tmux *Tmux) error { return _fnKeySplitWindow(tmux, "-h") }
func fnKeySplitWindowVertically(tmux *Tmux) error   { return _fnKeySplitWindow(tmux, "-v") }

func _fnKeySelectPane(tmux *Tmux, flag string) error {
	_, err := tmux.SendCommand([]byte("select-pane " + flag))
	go tmux.renderer.RefreshWindowList()
	//go tmux.updatePaneInfo("")
	return err
}
func fnKeySelectPaneUp(tmux *Tmux) error    { return _fnKeySelectPane(tmux, "-U") }
func fnKeySelectPaneDown(tmux *Tmux) error  { return _fnKeySelectPane(tmux, "-D") }
func fnKeySelectPaneLeft(tmux *Tmux) error  { return _fnKeySelectPane(tmux, "-L") }
func fnKeySelectPaneRight(tmux *Tmux) error { return _fnKeySelectPane(tmux, "-R") }
func fnKeySelectPaneLast(tmux *Tmux) error  { return _fnKeySelectPane(tmux, "-l") }

func fnKeyTilePanes(tmux *Tmux) error {
	_, err := tmux.SendCommand([]byte("select-layout -E"))
	go tmux.renderer.RefreshWindowList()
	return err
}

func _fnKeyResizePane(tmux *Tmux, flag string) error {
	_, err := tmux.SendCommand([]byte("resize-pane " + flag))
	go tmux.renderer.RefreshWindowList()
	//go tmux.updatePaneInfo("")
	return err
}
func fnKeyResizePaneUp1(tmux *Tmux) error    { return _fnKeyResizePane(tmux, "-U 1") }
func fnKeyResizePaneDown1(tmux *Tmux) error  { return _fnKeyResizePane(tmux, "-D 1") }
func fnKeyResizePaneLeft1(tmux *Tmux) error  { return _fnKeyResizePane(tmux, "-L 1") }
func fnKeyResizePaneRight1(tmux *Tmux) error { return _fnKeyResizePane(tmux, "-R 1") }
func fnKeyResizePaneUp5(tmux *Tmux) error    { return _fnKeyResizePane(tmux, "-U 5") }
func fnKeyResizePaneDown5(tmux *Tmux) error  { return _fnKeyResizePane(tmux, "-D 5") }
func fnKeyResizePaneLeft5(tmux *Tmux) error  { return _fnKeyResizePane(tmux, "-L 5") }
func fnKeyResizePaneRight5(tmux *Tmux) error { return _fnKeyResizePane(tmux, "-R 5") }

/*
	List Key Bindings
*/

func fnKeyListBindings(tmux *Tmux) error { // error only used because the func type signature required elsewhere
	var (
		hkList = hotkeys.List()
		slice  []string
	)

	for _, hk := range hkList {
		slice = append(slice, fmt.Sprintf("%-4s %-8s %s", hk.Prefix, hk.Hotkey, hk.Description))
	}

	selectCallback := func(i int) {
		prefix := codes.KeyName(strings.TrimSpace(slice[i][:4]))
		hotkey := codes.KeyName(strings.TrimSpace(slice[i][5 : 5+8]))
		err := hotkeys.KeyPressWithPrefix(prefix, hotkey)
		if err != nil {
			tmux.renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
		}
	}

	tmux.renderer.DisplayMenu("Hotkeys", slice, nil, selectCallback, nil)
	return nil
}

func (tmux *Tmux) ListKeyBindings() {
	_ = fnKeyListBindings(tmux)
}
