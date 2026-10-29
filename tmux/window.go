package tmux

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/utils/workspace"
)

/*
	window_active                1 if window active
	window_active_clients        Number of clients viewing this window
	window_active_clients_list   List of clients viewing this window
	window_active_sessions       Number of sessions on which this window is active
	window_active_sessions_list  List of sessions on which this window is active
	window_activity              Time of window last activity
	window_activity_flag         1 if window has activity
	window_bell_flag             1 if window has bell
	window_bigger                1 if window is larger than client
	window_cell_height           Height of each cell in pixels
	window_cell_width            Width of each cell in pixels
	window_end_flag              1 if window has the highest index
	window_flags             #F  Window flags with # escaped as ##
	window_format                1 if format is for a window
	window_height                Height of window
	window_id                    Unique window ID
	window_index             #I  Index of window
	window_last_flag             1 if window is the last used
	window_layout                Window layout description, ignoring zoomed window panes
	window_linked                1 if window is linked across sessions
	window_linked_sessions       Number of sessions this window is linked to
	window_linked_sessions_list  List of sessions this window is linked to
	window_marked_flag           1 if window contains the marked pane
	window_name              #W  Name of window
	window_offset_x              X offset into window if larger than client
	window_offset_y              Y offset into window if larger than client
	window_panes                 Number of panes in window
	window_raw_flags             Window flags with nothing escaped
	window_silence_flag          1 if window has silence alert
	window_stack_index           Index in session most recent stack
	window_start_flag            1 if window has the lowest index
	window_visible_layout        Window layout description, respecting zoomed window panes
	window_width                 Width of window
	window_zoomed_flag           1 if window is zoomed
*/

var CMD_LIST_WINDOWS = "list-windows"

type WindowT struct {
	name       string
	id         string
	index      int
	width      int
	height     int
	active     bool
	panes      paneMap
	activePane *PaneT
	closed     bool
}

func (tmux *Tmux) initSessionWindows() error {
	//windows, err := tmux.SendCommandWithReflection(CMD_LIST_WINDOWS, reflect.TypeOf(WindowT{}))
	//if err != nil {
	//	return err
	//}

	tmux.wins = newWindowMap() //make(map[string]*WindowT)

	return tmux.updateWinInfo("")

	//command := fmt.Sprintf("set-option -w -t %s window-size latest", win.Id)
	//_, _ = tmux.SendCommand([]byte(command))
	//if err != nil {
	//	return err
	//}

	//return nil
}

// NewWindow is a request sent to tmux to trigger a new window event
func (tmux *Tmux) NewWindow() {
	err := tmux.newWindowWithInitialEnvironment()
	if err != nil {
		tmux.renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
	}
}

func (tmux *Tmux) newWindowWithInitialEnvironment() error {
	environment := cloneEnvironment(tmux.initialEnvVars)
	if pane, err := tmux.ActivePane(); err == nil && pane != nil && pane.term != nil {
		environment = pane.term.GetEnvVars()
	}
	command := "new-window -P -F '#{pane_id}'" + workspace.EnvsTmux(environment)
	response, err := tmux.SendCommand([]byte(command))
	if err != nil {
		return err
	}
	tmux.seedPaneEnvironment(responsePaneID(response), environment)
	return nil
}

// newWindow is a new window event being returned from tmux
func (tmux *Tmux) newWindow(winId string, caller types.CallerT) *WindowT {
	win := &WindowT{
		id:    winId,
		panes: newPaneMap(),
	}

	tmux.wins.Set(winId, win)

	if caller != types.CALLER_updateWinInfo {
		// don't get caught in a loop!
		err := tmux.updateWinInfo(winId)
		if err != nil {
			tmux.renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
		}
	}

	return win
}

type winInfo struct {
	Name      string `tmux:"window_name"`
	Id        string `tmux:"window_id"`
	Index     int    `tmux:"window_index"`
	Width     int    `tmux:"window_width"`
	Height    int    `tmux:"window_height"`
	Active    bool   `tmux:"?window_active,true,false"`
	PaneCount int    `tmux:"window_panes"`
}

// updateWinInfo, winId is optional. Leave blank to update all windows
func (tmux *Tmux) updateWinInfo(winId string) error {
	var filter string
	if winId != "" {
		filter = fmt.Sprintf("-f '#{m:#{window_id},%s}'", winId)
	}

	v, err := tmux.SendCommandWithReflection(CMD_LIST_WINDOWS, reflect.TypeOf(winInfo{}), filter)
	if err != nil {
		return err
	}

	wins, ok := v.([]any)
	if !ok {
		return fmt.Errorf("expecting an array of windows, instead got %T", v)
	}

	if winId == "" {
		for win := range tmux.wins.Each() {
			win.closed = true
		}
	}

	for i := range wins {
		info, ok := wins[i].(*winInfo)
		if !ok {
			return fmt.Errorf("expecting info on a window, instead got %T", info)
		}

		win := tmux.wins.Get(info.Id)
		if win == nil {
			win = tmux.newWindow(info.Id, types.CALLER_updateWinInfo)
		}
		win.index = info.Index
		win.name = info.Name
		win.width = info.Width
		win.height = info.Height
		win.active = info.Active
		win.closed = info.PaneCount == 0

		if win.active {
			tmux.activeWindow = win
		}
	}

	return nil
}

func (tmux *Tmux) ActiveWindow() (*WindowT, error) {
	if tmux == nil {
		return nil, errors.New("no open windows")
	}

	var fallback *WindowT
	for win := range tmux.wins.Each() {
		if win == nil || win.closed {
			continue
		}
		if win.active {
			tmux.activeWindow = win
			return win, nil
		}
		if fallback == nil || win.index < fallback.index || (win.index == fallback.index && win.id < fallback.id) {
			fallback = win
		}
	}
	if cached := tmux.activeWindow; cached != nil && !cached.closed && tmux.wins.Get(cached.id) == cached {
		return cached, nil
	}
	tmux.activeWindow = fallback
	if fallback == nil {
		return nil, errors.New("no open windows")
	}
	return fallback, nil
}

func (win *WindowT) ActivePane() *PaneT {
	if win == nil || win.closed {
		return nil
	}
	var fallback *PaneT
	for pane := range win.panes.Each() {
		if pane == nil || pane.closed {
			continue
		}
		if pane.active {
			win.activePane = pane
			return pane
		}
		if fallback == nil || pane.top < fallback.top || (pane.top == fallback.top && pane.left < fallback.left) || (pane.top == fallback.top && pane.left == fallback.left && pane.id < fallback.id) {
			fallback = pane
		}
	}
	if cached := win.activePane; cached != nil && !cached.closed && win.panes.Get(cached.id) == cached {
		return cached
	}
	win.activePane = fallback
	return fallback
}

func (tmux *Tmux) RenameWindow(win *WindowT, name string) error {
	command := fmt.Sprintf("rename-window -t %s '%s'", win.id, name)
	_, err := tmux.SendCommand([]byte(command))
	return err
}

func (tmux *Tmux) SelectAndResizeWindow(winId string, size *types.XY) error {
	command := fmt.Sprintf("resize-window -t %s -x %d -y %d", winId, size.X, size.Y)
	_, err := tmux.SendCommand([]byte(command))
	if err != nil {
		return err
	}

	tmux.selectWindow(winId)

	win := tmux.wins.Get(winId)
	if win == nil {
		return fmt.Errorf("unable to get window with ID '%s'", winId)
	}

	for pane := range win.panes.Each() {
		go pane.Resize(&types.XY{X: int32(pane.width), Y: int32(pane.height)})
	}

	return err
}

func (tmux *Tmux) selectWindow(winId string) error {
	command := fmt.Sprintf("select-window -t %s", winId)
	_, err := tmux.SendCommand([]byte(command))

	// old window
	tmux.activeWindow.active = false

	// new window
	win := tmux.wins.Get(winId)
	if win == nil {
		return fmt.Errorf("unable to get window with ID '%s'", winId)
	}
	tmux.activeWindow = win
	tmux.activeWindow.active = true

	//go tmux.UpdateSession()

	return err
}

func (tmux *Tmux) CloseWindow(winId string) {
	win := tmux.wins.Get(winId)
	if win == nil {
		tmux.renderer.DisplayNotification(types.NOTIFY_ERROR, fmt.Sprintf("Cannot find window %s to close", winId))
		return
	}

	//win.closeAllPanes()
	win.close(tmux)
}

func (win *WindowT) close(tmux *Tmux) {
	win.closed = true
	//delete(tmux.wins, win.id)
	tmux.wins.Delete(win.id)

	msg := fmt.Sprintf("Closing window %s: %s", win.id, win.name)
	tmux.renderer.DisplayNotification(types.NOTIFY_INFO, msg)
}

/*func (win *WindowT) closeAllPanes() {
	//win.panes.mutex.Lock()
	for pane := range win.panes.Each() {
		pane.exit()
	}
	//win.panes.mutex.Unlock()
}
*/
