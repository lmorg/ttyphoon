package tmux

import (
	"fmt"

	"github.com/lmorg/mxtty/types"
)

/*
	session_activity              Time of session last activity
	session_alerts                List of window indexes with alerts
	session_attached              Number of clients session is attached to
	session_attached_list         List of clients session is attached to
	session_created               Time session created
	session_format                1 if format is for a session
	session_group                 Name of session group
	session_group_attached        Number of clients sessions in group are attached to
	session_group_attached_list   List of clients sessions in group are attached to
	session_group_list            List of sessions in group
	session_group_many_attached   1 if multiple clients attached to sessions in group
	session_group_size            Size of session group
	session_grouped               1 if session in a group
	session_id                    Unique session ID
	session_last_attached         Time session last attached
	session_many_attached         1 if multiple clients attached
	session_marked                1 if this session contains the marked pane
	session_name              #S  Name of session
	session_path                  Working directory of session
	session_stack                 Window indexes in most recent order
	session_windows               Number of windows in session
*/

const _PANE_EXITED = "Pane exited"

func (tmux *Tmux) setSessionHooks() error {
	command := fmt.Sprintf(`set-hook -g pane-exited 'display-message "%s"'`, _PANE_EXITED)
	_, err := tmux.SendCommand([]byte(command))
	return err
}

const _COMMAND_SET_OPTION = "set-option terminal-features[%d] %s"

var sessionSetOptions = []string{
	"256",         // Supports 256 colours with the SGR escape sequences.
	"//clipboard", // Allows setting the system clipboard.
	//"ccolour",       // Allows setting the cursor colour.
	//"cstyle",        // Allows setting the cursor style.
	"extkeys", // Supports extended keys.
	//"focus",         // Supports focus reporting.
	//"hyperlinks",    // Supports OSC 8 hyperlinks.
	"ignorefkeys", // Ignore function keys from terminfo(5) and use the tmux internal set only.
	//"margins",       // Supports DECSLRM margins.
	//"mouse",         // Supports xterm(1) mouse sequences.
	//"osc7",          // Supports the OSC 7 working directory extension.
	//"overline",      // Supports the overline SGR attribute.
	//"rectfill",      // Supports the DECFRA rectangle fill escape sequence.
	"RGB",           // Supports RGB colour with the SGR escape sequences.
	"sixel",         // Supports SIXEL graphics.
	"strikethrough", // Supports the strikethrough SGR escape sequence.
	//"sync",          // Supports synchronized updates.
	"title", // Supports xterm(1) title setting.
	//"usstyle",       // Allows underscore style and colour to be set.
}

func (tmux *Tmux) setSessionTerminalFeatures() error {
	for i := range sessionSetOptions {
		command := fmt.Sprintf(_COMMAND_SET_OPTION, i, sessionSetOptions[i])
		_, err := tmux.SendCommand([]byte(command))
		if err != nil {
			return err
		}
	}

	//_, err := tmux.SendCommand([]byte("set-window-option synchronize-panes on"))
	//return err
	return nil
}

func (tmux *Tmux) initSession(renderer types.Renderer, size *types.XY) error {
	err := tmux.setSessionTerminalFeatures()
	if err != nil {
		return err
	}

	err = tmux.RefreshClient(size)
	if err != nil {
		return err
	}

	err = tmux.initSessionWindows()
	if err != nil {
		return err
	}

	err = tmux.initSessionPanes(renderer)
	if err != nil {
		return err
	}

	err = tmux._getDefaultTmuxKeyBindings()
	if err != nil {
		return err
	}

	err = tmux.setSessionHooks()
	if err != nil {
		return err
	}

	tmux.ActivePane().term.MakeVisible(true)
	return nil
}

func (tmux *Tmux) UpdateSession() {
	err := tmux.updateWinInfo("")
	if err != nil {
		tmux.renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
	}

	err = tmux.updatePaneInfo("")
	if err != nil {
		tmux.renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
	}

	tmux.renderer.RefreshWindowList()
}
