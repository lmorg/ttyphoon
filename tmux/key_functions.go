package tmux

import (
	"fmt"
	"time"

	"github.com/lmorg/mxtty/types"
)

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
	windowNames := make([]string, len(tmux.appWindow.Tabs))
	for i := range tmux.appWindow.Tabs {
		windowNames[i] = tmux.appWindow.Tabs[i].Name()
	}

	_highlightCallback := func(i int) {
		if tmux.activeWindow.id == tmux.appWindow.Tabs[i].Id() {
			return
		}

		oldTerm := tmux.activeWindow.activePane.term
		err := tmux.SelectAndResizeWindow(tmux.appWindow.Tabs[i].Id(), tmux.renderer.GetWindowSizeCells())
		if err != nil {
			tmux.renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
		}

		win := tmux.wins.Get(tmux.appWindow.Tabs[i].Id())
		if win != nil {
			win.activePane.term.ShowCursor(false)
		}

		//windows[i].activePane.Term().ShowCursor(false)
		go func() {
			// this is a kludge to avoid the cursor showing as you switch windows
			time.Sleep(500 * time.Millisecond)
			oldTerm.ShowCursor(true)
		}()
	}

	_chooseCallback := func(i int) {
		err := tmux.SelectAndResizeWindow(tmux.appWindow.Tabs[i].Id(), tmux.renderer.GetWindowSizeCells())
		if err != nil {
			tmux.renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
		}
	}

	activeWindow := tmux.activeWindow.id
	_cancelCallback := func(_ int) {
		err := tmux.selectWindow(activeWindow)
		if err != nil {
			tmux.renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
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

func fnKeyListBindings(tmux *Tmux) error {
	/*	var slice []string
		for key, fn := range tmux.keys.fnTable {
			slice = append(slice, fmt.Sprintf("%-4s %-8s %s", tmux.keys.prefix, key, fn.note))
		}

		sort.Strings(slice)

		selectCallback := func(i int) {
			s := strings.TrimSpace(slice[i][5 : 5+8])
			err := tmux.keys.fnTable[s].fn(tmux)
			if err != nil {
				tmux.renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
			}
		}

		tmux.renderer.DisplayMenu("Hotkeys", slice, nil, selectCallback, nil)
		return nil*/
	return fmt.Errorf("TODO: I need to rewrite this again")
}

func (tmux *Tmux) ListKeyBindings() {
	_ = fnKeyListBindings(tmux)
}
