package tmux

import (
	"bytes"
	"fmt"
	"reflect"

	"github.com/lmorg/mxtty/codes"
	"github.com/lmorg/mxtty/debug"
	virtualterm "github.com/lmorg/mxtty/term"
	"github.com/lmorg/mxtty/types"
	runebuf "github.com/lmorg/mxtty/utils/rune_buf"
)

/*
	pane_active                1 if active pane
	pane_at_bottom             1 if pane is at the bottom of window
	pane_at_left               1 if pane is at the left of window
	pane_at_right              1 if pane is at the right of window
	pane_at_top                1 if pane is at the top of window
	pane_bg                    Pane background colour
	pane_bottom                Bottom of pane
	pane_current_command       Current command if available
	pane_current_path          Current path if available
	pane_dead                  1 if pane is dead
	pane_dead_signal           Exit signal of process in dead pane
	pane_dead_status           Exit status of process in dead pane
	pane_dead_time             Exit time of process in dead pane
	pane_fg                    Pane foreground colour
	pane_format                1 if format is for a pane
	pane_height                Height of pane
	pane_id                #D  Unique pane ID
	pane_in_mode               1 if pane is in a mode
	pane_index             #P  Index of pane
	pane_input_off             1 if input to pane is disabled
	pane_key_mode              Extended key reporting mode in this pane
	pane_last                  1 if last pane
	pane_left                  Left of pane
	pane_marked                1 if this is the marked pane
	pane_marked_set            1 if a marked pane is set
	pane_mode                  Name of pane mode, if any
	pane_path                  Path of pane (can be set by application)
	pane_pid                   PID of first process in pane
	pane_pipe                  1 if pane is being piped
	pane_right                 Right of pane
	pane_search_string         Last search string in copy mode
	pane_start_command         Command pane started with
	pane_start_path            Path pane started with
	pane_synchronized          1 if pane is synchronized
	pane_tabs                  Pane tab positions
	pane_title             #T  Title of pane (can be set by application)
	pane_top                   Top of pane
	pane_tty                   Pseudo terminal of pane
	pane_unseen_changes        1 if there were changes in pane while in mode
	pane_width                 Width of pane
*/

var CMD_LIST_PANES = "list-panes"

type PaneT struct {
	title    string
	id       string
	width    int
	height   int
	left     int32
	top      int32
	right    int32
	bottom   int32
	active   bool
	atBottom bool
	windowId string
	curPath  string
	tmux     *Tmux
	term     types.Term
	buf      *runebuf.Buf
	closed   bool
}

func (tmux *Tmux) initSessionPanes(renderer types.Renderer) error {
	v, err := tmux.SendCommandWithReflection(CMD_LIST_PANES, reflect.TypeOf(paneInfo{}), "-s")
	if err != nil {
		return err
	}

	panes, ok := v.([]any)
	if !ok {
		return fmt.Errorf("expecting an array of panes, instead got %T", v)
	}

	for i := range panes {
		info, ok := panes[i].(*paneInfo)
		if !ok {
			return fmt.Errorf("expecting info on a pane, instead got %T", info)
		}

		pane := info.updatePane(tmux)

		command := fmt.Sprintf("capture-pane -J -S- -E- -e -p -t %s", pane.id)
		resp, err := tmux.SendCommand([]byte(command))
		if err != nil {
			renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
		} else {
			b := bytes.Join(resp.Message, []byte{'\r', '\n'}) // CRLF
			pane.buf.Write(b)
		}

		command = fmt.Sprintf(`display-message -p -t %s "#{e|+:#{cursor_y},1};#{e|+:#{cursor_x},1}H"`, pane.id)
		resp, err = tmux.SendCommand([]byte(command))
		if err != nil {
			renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
		} else {
			b := append([]byte{codes.AsciiEscape, '['}, resp.Message[0]...)
			pane.buf.Write(b)
		}
	}

	return nil
}

func (tmux *Tmux) newPane(info *paneInfo) *PaneT {
	debug.Log(info)

	pane := &PaneT{
		id:   info.Id,
		tmux: tmux,
		buf:  runebuf.New(),
	}

	virtualterm.NewTerminal(
		pane, tmux.renderer,
		&types.XY{X: int32(info.Width), Y: int32(info.Height)},
		false)

	pane.term.Start(pane)

	tmux.panes.Set(pane.id, pane)

	return pane
}

type paneInfo struct {
	Id        string `tmux:"pane_id"`
	Title     string `tmux:"pane_title"`
	Width     int    `tmux:"pane_width"`
	Height    int    `tmux:"pane_height"`
	PosLeft   int    `tmux:"pane_left"`
	PosTop    int    `tmux:"pane_top"`
	PosRight  int    `tmux:"pane_right"`
	PosBottom int    `tmux:"pane_bottom"`
	Active    bool   `tmux:"?pane_active,true,false"`
	Dead      bool   `tmux:"?pane_dead,true,false"`
	WindowId  string `tmux:"window_id"`
	WinActive bool   `tmux:"?window_active,true,false"`
	AtBottom  bool   `tmux:"?pane_at_bottom,true,false"`
	CurPath   string `tmux:"pane_current_path"`
}

// updatePaneInfo, paneId is optional. Leave blank to update all panes
func (tmux *Tmux) updatePaneInfo(paneId string) error {
	var filter string
	if paneId != "" {
		filter = fmt.Sprintf("-f '#{m:#{pane_id},%s}'", paneId)
	}

	v, err := tmux.SendCommandWithReflection(CMD_LIST_PANES, reflect.TypeOf(paneInfo{}), "-s", filter)
	if err != nil {
		return err
	}

	panes, ok := v.([]any)
	if !ok {
		return fmt.Errorf("expecting an array of panes, instead got %T", v)
	}

	for i := range panes {
		info, ok := panes[i].(*paneInfo)
		if !ok {
			return fmt.Errorf("expecting info on a pane, instead got %T", info)
		}

		info.updatePane(tmux)
	}

	return nil
}

func (info *paneInfo) updatePane(tmux *Tmux) *PaneT {
	pane := tmux.panes.Get(info.Id)
	if pane == nil {
		pane = tmux.newPane(info)
	}

	/*if info.Dead {
		pane.Close()
		continue
	}*/

	if pane.closed {
		return pane
	}

	pane.title = info.Title
	pane.width = info.Width
	pane.height = info.Height
	pane.active = info.Active
	pane.windowId = info.WindowId
	pane.left = int32(info.PosLeft)
	pane.top = int32(info.PosTop)
	pane.right = int32(info.PosRight)
	pane.bottom = int32(info.PosBottom)
	pane.atBottom = info.AtBottom
	pane.curPath = info.CurPath
	if pane.term != nil {
		pane.term.MakeVisible(info.WinActive)
		pane.term.HasFocus(info.Active)
		pane.term.Resize(&types.XY{X: int32(info.Width), Y: int32(info.Height)})
	}

	win, ok := tmux.wins[pane.windowId]
	if !ok {
		/*err := tmux.updateWinInfo(pane.WindowId)
		if err != nil {
			panic(err)
		}
		win = tmux.win[pane.WindowId]*/
		panic("tmux pane created before window")
	}
	win.panes.Set(pane.id, pane)
	if pane.active {
		win.activePane = pane
	}

	return pane
}

func (tmux *Tmux) ActivePane() *PaneT {
	return tmux.ActiveWindow().ActivePane()
}

func (tmux *Tmux) SelectPane(paneId string) error {
	command := fmt.Sprintf("select-pane -t %s", paneId)
	_, err := tmux.SendCommand([]byte(command))

	//go tmux.UpdateSession()

	return err
}

func (tmux *Tmux) paneExited() error {
	v, err := tmux.SendCommandWithReflection(CMD_LIST_PANES, reflect.TypeOf(paneInfo{}), "-s")
	if err != nil {
		return err
	}

	panes, ok := v.([]any)
	if !ok {
		return fmt.Errorf("expecting an array of panes, instead got %T", v)
	}

	// start bypass the paneMap helper functions
	//tmux.panes.mutex.Lock()
	for pane := range tmux.panes.Each() {
		pane.closed = true
	}
	//tmux.panes.mutex.Unlock()
	// end bypass the paneMap helper functions

	for i := range panes {
		info, ok := panes[i].(*paneInfo)
		if !ok {
			return fmt.Errorf("expecting info on a pane, instead got %T", info)
		}

		if pane := tmux.panes.Get(info.Id); pane != nil {
			pane.closed = false
		}
	}

	for pane := range tmux.panes.Each() {
		if pane.closed {
			pane.exit()
		}
	}

	go tmux.renderer.RefreshWindowList()

	return nil
}
