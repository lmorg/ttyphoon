package tmux

import (
	"fmt"
	"reflect"
	"sort"

	"github.com/lmorg/mxtty/debug"
	"github.com/lmorg/mxtty/types"
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
	Name       string `tmux:"window_name"`
	Id         string `tmux:"window_id"`
	Index      int    `tmux:"window_index"`
	Width      int    `tmux:"window_width"`
	Height     int    `tmux:"window_height"`
	Active     bool   `tmux:"?window_active,true,false"`
	panes      map[string]*PaneT
	activePane *PaneT
	closed     bool
}

func (tmux *Tmux) initSessionWindows() error {
	windows, err := tmux.sendCommand(CMD_LIST_WINDOWS, reflect.TypeOf(WindowT{}))
	if err != nil {
		return err
	}

	tmux.win = make(map[string]*WindowT)

	for i := range windows.([]any) {
		win := windows.([]any)[i].(*WindowT)
		win.panes = make(map[string]*PaneT)
		tmux.win[win.Id] = win
		if win.Active {
			tmux.activeWindow = win
		}

		command := fmt.Sprintf("set-option -w -t %s window-size latest", win.Id)
		_, _ = tmux.SendCommand([]byte(command))
		//if err != nil {
		//	return err
		//}
	}

	debug.Log(windows.([]any))
	return nil
}

func (tmux *Tmux) newWindow(winId string) *WindowT {
	win := &WindowT{
		Id:    winId,
		panes: make(map[string]*PaneT),
	}

	tmux.win[winId] = win
	//tmux.activeWindow = win
	return win
}

type winInfo struct {
	Id     string `tmux:"window_id"`
	Index  int    `tmux:"window_index"`
	Name   string `tmux:"window_name"`
	Width  int    `tmux:"window_width"`
	Height int    `tmux:"window_height"`
	Active bool   `tmux:"?window_active,true,false"`
}

// updateWinInfo, winId is optional. Leave blank to update all windows
func (tmux *Tmux) updateWinInfo(winId string) error {
	var filter string
	if winId != "" {
		filter = fmt.Sprintf("-f '#{m:#{window_id},%s}'", winId)
	}

	v, err := tmux.sendCommand(CMD_LIST_WINDOWS, reflect.TypeOf(winInfo{}), filter)
	if err != nil {
		return err
	}

	wins, ok := v.([]any)
	if !ok {
		return fmt.Errorf("expecting an array of windows, instead got %T", v)
	}

	for i := range wins {

		info, ok := wins[i].(*winInfo)
		if !ok {
			return fmt.Errorf("expecting info on a window, instead got %T", info)
		}

		win, ok := tmux.win[info.Id]
		if !ok {
			win = tmux.newWindow(info.Id)
		}
		win.Index = info.Index
		win.Name = info.Name
		win.Width = info.Width
		win.Height = info.Height
		win.Active = info.Active

		if win.Active {
			tmux.activeWindow = win
		}
	}

	return nil
}

func (tmux *Tmux) RenderWindows() []*WindowT {
	var wins []*WindowT

	for _, win := range tmux.win {
		if win.closed {
			continue
		}
		wins = append(wins, win)
	}

	sort.Slice(wins, func(i, j int) bool {
		return wins[i].Index < wins[j].Index
	})

	return wins
}

func (tmux *Tmux) ActiveWindow() *types.TermWindow {
	_ = tmux.updatePaneInfo("")

	tw := new(types.TermWindow)

	for _, pane := range tmux.activeWindow.panes {
		if pane.closed {
			debug.Log(fmt.Sprintf("skipping closed pane %s", pane.Id))
			continue
		}
		tw.Tiles = append(tw.Tiles, pane.tile)
	}

	tw.Active = tmux.activeWindow.ActivePane().tile

	debug.Log(tw)

	return tw
}

func (win *WindowT) ActivePane() *PaneT {
	if !win.activePane.closed {
		return win.activePane
	}

	err := fnKeySelectPaneLast(win.activePane.tmux)
	if err == nil && !win.activePane.closed {
		return win.activePane
	}

	err = fnKeySelectPaneUp(win.activePane.tmux)
	if err == nil && !win.activePane.closed {
		return win.activePane
	}

	err = fnKeySelectPaneLeft(win.activePane.tmux)
	if err == nil && !win.activePane.closed {
		return win.activePane
	}

	if err != nil {
		win.activePane.tmux.renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
	} else {
		win.activePane.tmux.renderer.DisplayNotification(types.NOTIFY_ERROR, "Cannot find an active pane")
	}

	return win.activePane
}

func (win *WindowT) Rename(name string) error {
	command := fmt.Sprintf("rename-window -t %s '%s'", win.Id, name)
	_, err := win.activePane.tmux.SendCommand([]byte(command))
	return err
}

func (tmux *Tmux) SelectAndResizeWindow(winId string, size *types.XY) error {
	command := fmt.Sprintf("resize-window -t %s -x %d -y %d", winId, size.X, size.Y)
	_, err := tmux.SendCommand([]byte(command))
	if err != nil {
		return err
	}

	tmux.selectWindow(winId)

	for _, pane := range tmux.win[winId].panes {
		_ = pane.Resize(&types.XY{X: int32(pane.Width), Y: int32(pane.Height)})
	}

	return err
}

func (tmux *Tmux) selectWindow(winId string) error {
	command := fmt.Sprintf("select-window -t %s", winId)
	_, err := tmux.SendCommand([]byte(command))

	// old window
	tmux.activeWindow.Active = false

	// new window
	tmux.activeWindow = tmux.win[winId]
	tmux.activeWindow.Active = true

	go tmux.UpdateSession()

	return err
}

func (win *WindowT) Close() {
	win.closed = true
	go win.activePane.tmux.UpdateSession()
	for _, pane := range win.panes {
		pane.Close()
	}
}
