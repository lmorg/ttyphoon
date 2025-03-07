package tmux

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/lmorg/mxtty/types"
)

func fnKeyNewWindow(tmux *Tmux) error {
	_, err := tmux.SendCommand([]byte("new-window"))
	return err
}

func fnKeyKillPane(tmux *Tmux) error {
	command := fmt.Sprintf("kill-pane -t %s", tmux.ActivePane().Id)
	_, err := tmux.SendCommand([]byte(command))
	return err
}

func fnKeyKillCurrentWindow(tmux *Tmux) error {
	command := fmt.Sprintf("kill-window -t %s", tmux.activeWindow.Id)
	_, err := tmux.SendCommand([]byte(command))
	return err
}

func fnKeyRenameWindow(tmux *Tmux) error {
	tmux.renderer.DisplayInputBox("Please enter a new name for this window", tmux.activeWindow.Name, func(name string) {
		err := tmux.activeWindow.Rename(name)
		if err != nil {
			tmux.renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
		}
	})
	return nil
}

func fnKeyChooseWindowFromList(tmux *Tmux) error {
	windows := tmux.RenderWindows()

	windowNames := make([]string, len(windows))
	for i := range windows {
		windowNames[i] = windows[i].Name
	}

	_highlightCallback := func(i int) {
		if tmux.activeWindow.Id == windows[i].Id {
			return
		}

		oldTerm := tmux.activeWindow.activePane.Term()
		err := tmux.SelectAndResizeWindow(windows[i].Id, tmux.renderer.GetWindowSizeCells())
		if err != nil {
			tmux.renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
		}
		windows[i].activePane.Term().ShowCursor(false)
		go func() {
			// this is a kludge to avoid the cursor showing as you switch windows
			time.Sleep(500 * time.Millisecond)
			oldTerm.ShowCursor(true)
		}()
	}

	_chooseCallback := func(i int) {
		err := tmux.SelectAndResizeWindow(windows[i].Id, tmux.renderer.GetWindowSizeCells())
		if err != nil {
			tmux.renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
		}
	}

	activeWindow := tmux.activeWindow.Id
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
	wins := tmux.RenderWindows()
	if i >= len(wins) {
		return fmt.Errorf("there is not a window %d", i)
	}

	return tmux.selectWindow(wins[i].Id)
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
	go tmux.renderer.RefreshWindowList()
	return err
}
func fnKeySplitWindowHorizontally(tmux *Tmux) error { return _fnKeySplitWindow(tmux, "-h") }
func fnKeySplitWindowVertically(tmux *Tmux) error   { return _fnKeySplitWindow(tmux, "-v") }

func _fnKeySelectPane(tmux *Tmux, flag string) error {
	_, err := tmux.SendCommand([]byte("select-pane " + flag))
	go tmux.renderer.RefreshWindowList()
	return err
}
func fnKeySelectPaneUp(tmux *Tmux) error    { return _fnKeySelectPane(tmux, "-U") }
func fnKeySelectPaneDown(tmux *Tmux) error  { return _fnKeySelectPane(tmux, "-D") }
func fnKeySelectPaneLeft(tmux *Tmux) error  { return _fnKeySelectPane(tmux, "-L") }
func fnKeySelectPaneRight(tmux *Tmux) error { return _fnKeySelectPane(tmux, "-R") }
func fnKeySelectPaneLast(tmux *Tmux) error  { return _fnKeySelectPane(tmux, "-l") }

func fnKeyTilePanes(tmux *Tmux) error {
	_, err := tmux.SendCommand([]byte("select-layout -E"))
	//go errToNotification(tmux.renderer,
	//	tmux.SelectAndResizeWindow(tmux.activeWindow.Id, tmux.renderer.GetWindowSizeCells()))
	go tmux.renderer.RefreshWindowList()
	return err
}

func fnKeyListBindings(tmux *Tmux) error {
	var slice []string
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
	return nil
}

func (tmux *Tmux) ListKeyBindings() {
	_ = fnKeyListBindings(tmux)
}
