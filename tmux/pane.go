package tmux

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync/atomic"

	"github.com/lmorg/ttyphoon/codes"
	"github.com/lmorg/ttyphoon/debug"
	"github.com/lmorg/ttyphoon/types"
	runebuf "github.com/lmorg/ttyphoon/utils/rune_buf"
	"github.com/lmorg/ttyphoon/utils/workspace"
	"github.com/lmorg/ttyphoon/virtualterm"
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
	title        string
	id           string
	width        int
	height       int
	left         int32
	top          int32
	right        int32
	bottom       int32
	active       bool
	atBottom     bool
	zoomed       bool
	windowId     string
	pwd          string
	tmux         *Tmux
	term         types.Term
	buf          *runebuf.Buf
	closed       bool
	dead         atomic.Bool
	actionResult atomic.Int32
	autoCloseAt  atomic.Int64
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
		if pane == nil {
			continue
		}

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
	initialEnvVars := cloneEnvironment(tmux.initialEnvVars)
	tmux.pendingPaneEnvMu.Lock()
	if pending, ok := tmux.pendingPaneEnv[info.Id]; ok {
		initialEnvVars = cloneEnvironment(pending)
		delete(tmux.pendingPaneEnv, info.Id)
	}
	tmux.pendingPaneEnvMu.Unlock()

	pane := &PaneT{
		id:   info.Id,
		tmux: tmux,
		buf:  runebuf.New(),
		pwd:  info.workingDirectory("", true),
	}

	term := virtualterm.NewTerminal(
		pane, tmux.renderer,
		&types.XY{X: int32(info.Width), Y: int32(info.Height)},
		false)
	term.SetInitialEnvVars(initialEnvVars)

	pane.term.Start(pane)

	tmux.panes.Set(pane.id, pane)
	tmux.pendingPaneEnvMu.Lock()
	pendingEnvironment, hasPendingEnvironment := tmux.pendingPaneEnv[pane.id]
	if hasPendingEnvironment {
		delete(tmux.pendingPaneEnv, pane.id)
	}
	tmux.pendingPaneEnvMu.Unlock()
	if hasPendingEnvironment {
		pane.setInitialEnvVars(pendingEnvironment)
	}

	return pane
}

func (pane *PaneT) setInitialEnvVars(environment map[string]string) {
	if pane == nil || pane.term == nil {
		return
	}
	if setter, ok := pane.term.(interface{ SetInitialEnvVars(map[string]string) }); ok {
		setter.SetInitialEnvVars(environment)
	}
}

type paneInfo struct {
	Id        string `tmux:"pane_id"`
	Title     string //`tmux:"pane_title"`
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
	Path      string `tmux:"pane_current_path"`
	PathStart string `tmux:"pane_start_path"`
	Zoomed    bool   `tmux:"?pane_zoomed_flag,true,false"`
}

func (info *paneInfo) workingDirectory(previous string, isNew bool) string {
	if !isNew && !info.Dead && info.Path != "" {
		return info.Path
	}
	if previous != "" {
		return previous
	}
	if info.PathStart != "" {
		return info.PathStart
	}
	return info.Path
}

// updatePaneInfo, paneId is optional. Leave blank to update all panes
func (tmux *Tmux) updatePaneInfo(paneId string) error {
	var filter string
	if paneId != "" {
		filter = fmt.Sprintf("-f '#{m:#{pane_id},%s}'", paneId)
	}

	v, err := tmux.SendCommandWithReflection(CMD_LIST_PANES, reflect.TypeFor[paneInfo](), "-s", filter)
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
	var isNew bool
	win := tmux.wins.Get(info.WindowId)
	if win == nil || win.closed {
		return nil
	}

	pane := tmux.panes.Get(info.Id)
	if pane == nil {
		isNew = true
		pane = tmux.newPane(info)
	}

	if pane.closed {
		return pane
	}

	// we only see title escape codes emitted while attached, so seed from
	// tmux's own record for panes that predate us
	if pane.title == "" {
		pane.title = info.Title
	}
	pane.width = info.Width
	pane.height = info.Height
	pane.active = info.Active
	pane.windowId = info.WindowId
	pane.left = int32(info.PosLeft)
	pane.top = int32(info.PosTop)
	pane.right = int32(info.PosRight)
	pane.bottom = int32(info.PosBottom)
	pane.atBottom = info.AtBottom
	pane.zoomed = info.Zoomed
	pane.pwd = info.workingDirectory(pane.pwd, isNew)
	if pane.dead.Swap(info.Dead) && !info.Dead {
		pane.actionResult.Store(0)
		pane.autoCloseAt.Store(0)
	}
	win.panes.Set(pane.id, pane)
	if pane.active {
		win.activePane = pane
	}
	if pane.term != nil {
		pane.term.MakeVisible(info.WinActive)
		pane.term.SetFocus(info.Active)
		pane.term.Resize(&types.XY{X: int32(info.Width), Y: int32(info.Height)})
	}

	return pane
}

func (tmux *Tmux) ActivePane() (*PaneT, error) {
	if tmux == nil {
		return nil, fmt.Errorf("no open windows")
	}

	win, err := tmux.ActiveWindow()
	if err != nil {
		return nil, err
	}

	pane := win.ActivePane()
	if pane == nil {
		return nil, fmt.Errorf("no active pane available")
	}
	return pane, nil
}

func (tmux *Tmux) SelectPane(paneId string) error {
	command := fmt.Sprintf("select-pane -t %s", paneId)
	_, err := tmux.SendCommand([]byte(command))

	//go tmux.UpdateSession()

	return err
}

func (tmux *Tmux) ZoomPane(paneId string) error {
	pane := tmux.panes.Get(paneId)
	if pane == nil || pane.closed {
		return fmt.Errorf("pane not found")
	}
	command := fmt.Sprintf("resize-pane -t %s -Z", pane.id)
	_, err := tmux.SendCommand([]byte(command))

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

	livePanes := make(map[string]struct{}, len(panes))
	for i := range panes {
		info, ok := panes[i].(*paneInfo)
		if !ok {
			return fmt.Errorf("expecting info on a pane, instead got %T", info)
		}
		livePanes[info.Id] = struct{}{}
	}

	tmux.removeExitedPanes(livePanes)
	go tmux.renderer.RefreshWindowList()

	return nil
}

func (tmux *Tmux) removeExitedPanes(livePanes map[string]struct{}) {
	for pane := range tmux.panes.Each() {
		if _, ok := livePanes[pane.id]; !ok {
			pane.exit()
		}
	}
}

func (tmux *Tmux) ClonePane(commandLine string, callback interactiveActionCallback) error {
	pane, err := tmux.ActivePane()
	if err != nil {
		return err
	}
	if pane == nil || pane.closed || pane.term == nil {
		return fmt.Errorf("source terminal pane is unavailable")
	}
	if commandLine != "" && callback == nil {
		return fmt.Errorf("interactive action callback is required")
	}

	pwd := pane.pwd

	var actionID, scriptPath string
	if commandLine != "" {
		var err error
		actionID, err = newInteractiveActionID()
		if err != nil {
			return err
		}
		scriptPath, err = writeBashScript(commandLine, actionID, pwd)
		if err != nil {
			return err
		}
		if !tmux.registerInteractiveAction(actionID, scriptPath, callback) {
			_ = os.Remove(scriptPath)
			return fmt.Errorf("interactive action UUID collision")
		}
	}

	setupFailed := true
	newPaneID := ""
	defer func() {
		if setupFailed {
			if actionID != "" {
				tmux.discardInteractiveAction(actionID)
			}
			if newPaneID != "" {
				_, _ = tmux.SendCommand(fmt.Appendf(nil, "kill-pane -t %s", newPaneID))
			}
		}
	}()

	env := pane.term.GetEnvVars()

	tmuxCommand := clonePaneCommand(pane.id, pwd, env, scriptPath)
	response, err := tmux.SendCommand([]byte(tmuxCommand))
	if err != nil {
		return err
	}
	newPaneID = responsePaneID(response)
	if newPaneID == "" {
		return fmt.Errorf("tmux did not return the cloned pane ID")
	}
	tmux.seedPaneEnvironment(newPaneID, env)
	if err := tmux.updatePaneInfo(newPaneID); err != nil {
		return err
	}
	clonedPane := tmux.panes.Get(newPaneID)
	if clonedPane == nil || clonedPane.term == nil {
		return fmt.Errorf("cloned terminal pane %s was not initialized", newPaneID)
	}
	if tmux.renderer != nil {
		tmux.renderer.RefreshWindowList()
	}
	if actionID != "" && !tmux.bindInteractiveActionPane(actionID, newPaneID) {
		return fmt.Errorf("interactive action callback %s could not bind to pane %s", actionID, newPaneID)
	}
	setupFailed = false
	return nil
}

func quoteShellArgument(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func clonePaneCommand(paneID string, pwd string, env map[string]string, scriptPath string) string {
	keepExitedPane, command, workingDirectory := "", "", ""

	environmentArgs := workspace.EnvsTmux(env)

	//if pwd == "" {
	workingDirectory = `"#{pane_current_path}"`
	//} else {
	//	workingDirectory = workspace.QuoteString(pwd)
	//}

	if scriptPath != "" {
		keepExitedPane = " -k"
		command = " bash " + workspace.QuoteString(scriptPath)
	}

	return fmt.Sprintf(`split-window -v%s -P -F '#{pane_id}' -t %s -c %s%s%s`, keepExitedPane, paneID, workingDirectory, environmentArgs, command)
}

func responsePaneID(response *tmuxResponseT) string {
	if response == nil {
		return ""
	}
	for _, line := range response.Message {
		paneID := strings.TrimSpace(string(line))
		if strings.HasPrefix(paneID, "%") {
			return paneID
		}
	}
	return ""
}

func (pane *PaneT) HandleInteractiveActionCompletion(actionID string, exitCode int, output []byte) {
	if pane == nil || pane.tmux == nil {
		return
	}
	pane.tmux.completeInteractiveAction(actionID, pane.id, output, exitCode)
}

func (pane *PaneT) CommandFinished() bool {
	return pane != nil && (pane.dead.Load() || pane.actionResult.Load() != 0)
}

func (pane *PaneT) AutoCloseAt() int64 {
	if pane == nil {
		return 0
	}
	return max(0, pane.autoCloseAt.Load())
}

func (tmux *Tmux) CancelPaneAutoClose(paneID string) error {
	pane := tmux.panes.Get(paneID)
	if pane == nil || pane.closed {
		return fmt.Errorf("pane not found")
	}
	for {
		deadline := pane.autoCloseAt.Load()
		if deadline < 0 {
			return fmt.Errorf("pane is already closing")
		}
		if deadline == 0 || pane.autoCloseAt.CompareAndSwap(deadline, 0) {
			if tmux.renderer != nil {
				tmux.renderer.TriggerRedraw()
			}
			return nil
		}
	}
}

func (tmux *Tmux) CloseCompletedPane(paneID string) error {
	pane := tmux.panes.Get(paneID)
	if pane == nil || pane.closed {
		return fmt.Errorf("pane not found")
	}
	if !pane.CommandFinished() {
		return fmt.Errorf("pane command is still running")
	}
	return tmux.KillPane(paneID)
}

func (tmux *Tmux) PaneCommandFinished(paneID string) (bool, error) {
	pane := tmux.panes.Get(paneID)
	if pane == nil || pane.closed {
		return false, fmt.Errorf("pane not found")
	}
	return pane.CommandFinished(), nil
}

func (tmux *Tmux) KillPane(paneID string) error {
	pane := tmux.panes.Get(paneID)
	if pane == nil || pane.closed {
		return fmt.Errorf("pane not found")
	}
	if _, err := tmux.SendCommand(fmt.Appendf(nil, "kill-pane -t %s", pane.id)); err != nil {
		return err
	}
	return tmux.paneExited()
}

func (pane *PaneT) InteractiveActionChrome() *types.Colour {
	if pane == nil {
		return nil
	}
	switch pane.actionResult.Load() {
	case 1:
		return types.COLOR_OK
	case 2:
		return types.COLOR_ERROR
	default:
		return nil
	}
}
