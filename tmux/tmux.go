package tmux

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"

	"github.com/creack/pty"
	"github.com/lmorg/mxtty/config"
	"github.com/lmorg/mxtty/debug"
	"github.com/lmorg/mxtty/types"
	"github.com/lmorg/mxtty/utils/exit"
	"github.com/lmorg/mxtty/utils/octal"
)

/*
	Reference documentation used:
	- tmux man page: https://man.openbsd.org/tmux#CONTROL_MODE

	*** Control Mode: ***

	tmux offers a textual interface called control mode. This allows applications to communicate with tmux using a simple text-only protocol.

	In control mode, a client sends tmux commands or command sequences terminated by newlines on standard input. Each command will produce one block of output on standard output. An output block consists of a %begin line followed by the output (which may be empty). The output block ends with a %end or %error. %begin and matching %end or %error have three arguments: an integer time (as seconds from epoch), command number and flags (currently not used). For example:

	%begin 1363006971 2 1
	0: ksh* (1 panes) [80x24] [layout b25f,80x24,0,0,2] @2 (active)
	%end 1363006971 2 1

	The refresh-client -C command may be used to set the size of a client in control mode.

	In control mode, tmux outputs notifications. A notification will never occur inside an output block.

	The following notifications are defined:

	%client-detached client
		The client has detached.
	%client-session-changed client session-id name
		The client is now attached to the session with ID session-id, which is named name.
	%config-error error
		An error has happened in a configuration file.
	%continue pane-id
		The pane has been continued after being paused (if the pause-after flag is set, see refresh-client -A).
	%exit [reason]
		The tmux client is exiting immediately, either because it is not attached to any session or an error occurred. If present, reason describes why the client exited.
	%extended-output pane-id age ... : value
		New form of %output sent when the pause-after flag is set. age is the time in milliseconds for which tmux had buffered the output before it was sent. Any subsequent arguments up until a single ‘:’ are for future use and should be ignored.
	%layout-change window-id window-layout window-visible-layout window-flags
		The layout of a window with ID window-id changed. The new layout is window-layout. The window's visible layout is window-visible-layout and the window flags are window-flags.
	%message message
		A message sent with the display-message command.
	%output pane-id value
		A window pane produced output. value escapes non-printable characters and backslash as octal \xxx.
	%pane-mode-changed pane-id
		The pane with ID pane-id has changed mode.
	%paste-buffer-changed name
		Paste buffer name has been changed.
	%paste-buffer-deleted name
		Paste buffer name has been deleted.
	%pause pane-id
		The pane has been paused (if the pause-after flag is set).
	%session-changed session-id name
		The client is now attached to the session with ID session-id, which is named name.
	%session-renamed name
		The current session was renamed to name.
	%session-window-changed session-id window-id
		The session with ID session-id changed its active window to the window with ID window-id.
	%sessions-changed
		A session was created or destroyed.
	%subscription-changed name session-id window-id window-index pane-id ... : value
		The value of the format associated with subscription name has changed to value. See refresh-client -B. Any arguments after pane-id up until a single ‘:’ are for future use and should be ignored.
	%unlinked-window-add window-id
		The window with ID window-id was created but is not linked to the current session.
	%unlinked-window-close window-id
		The window with ID window-id, which is not linked to the current session, was closed.
	%unlinked-window-renamed window-id
		The window with ID window-id, which is not linked to the current session, was renamed.
	%window-add window-id
		The window with ID window-id was linked to the current session.
	%window-close window-id
		The window with ID window-id closed.
	%window-pane-changed window-id pane-id
		The active pane in the window with ID window-id changed to the pane with ID pane-id.
	%window-renamed window-id name
		The window with ID window-id was renamed to name.
		All the notifications listed in the CONTROL MODE section are hooks (without any arguments), except %exit. The following additional hooks are available:
*/

var (
	_RESP_OUTPUT = []byte("%output")
	_RESP_BEGIN  = []byte("%begin")
	_RESP_END    = []byte("%end")
	_RESP_ERROR  = []byte("%error")

	_RESP_CLIENT_DETACHED         = []byte("%client-detached")
	_RESP_CLIENT_SESSION_CHANGED  = []byte("%client-session-changed")
	_RESP_CONFIG_ERROR            = []byte("%config-error")
	_RESP_CONTINUE                = []byte("%continue")
	_RESP_EXIT                    = []byte("%exit")
	_RESP_EXTENDED_OUTPUT         = []byte("%extended-output")
	_RESP_LAYOUT_CHANGE           = []byte("%layout-change")
	_RESP_MESSAGE                 = []byte("%message")
	_RESP_PANE_MODE_CHANGED       = []byte("%pane-mode-changed")
	_RESP_PASTE_BUFFER_CHANGED    = []byte("%paste-buffer-changed")
	_RESP_PASTE_BUFFER_DELETED    = []byte("%paste-buffer-deleted")
	_RESP_PAUSE                   = []byte("%pause")
	_RESP_SESSION_CHANGED         = []byte("%session-changed")
	_RESP_SESSION_RENAMED         = []byte("%session-renamed")
	_RESP_SESSION_WINDOW_CHANGED  = []byte("%session-window-changed")
	_RESP_SESSIONS_CHANGED        = []byte("%sessions-changed")
	_RESP_SUBSCRIPTION_CHANGED    = []byte("%subscription-changed")
	_RESP_UNLINKED_WINDOW_ADD     = []byte("%unlinked-window-add")
	_RESP_UNLINKED_WINDOW_CLOSE   = []byte("%unlinked-window-close")
	_RESP_UNLINKED_WINDOW_RENAMED = []byte("%unlinked-window-renamed")
	_RESP_WINDOW_ADD              = []byte("%window-add")
	_RESP_WINDOW_CLOSE            = []byte("%window-close")
	_RESP_WINDOW_PANE_CHANGED     = []byte("%window-pane-changed")
	_RESP_WINDOW_RENAMED          = []byte("%window-renamed")
)

var respIgnored = [][]byte{
	_RESP_CLIENT_DETACHED,
	_RESP_CLIENT_SESSION_CHANGED,
	_RESP_CONFIG_ERROR,
	_RESP_CONTINUE,
	_RESP_EXTENDED_OUTPUT,
	_RESP_LAYOUT_CHANGE,
	_RESP_PANE_MODE_CHANGED,
	_RESP_PASTE_BUFFER_CHANGED,
	_RESP_PASTE_BUFFER_DELETED,
	_RESP_PAUSE,
	_RESP_SESSION_CHANGED,
	_RESP_SESSION_RENAMED,
	_RESP_SESSION_WINDOW_CHANGED,
	_RESP_SESSIONS_CHANGED,
	_RESP_SUBSCRIPTION_CHANGED,
	_RESP_UNLINKED_WINDOW_ADD,
	_RESP_UNLINKED_WINDOW_RENAMED,
	_RESP_WINDOW_PANE_CHANGED,
}

type Tmux struct {
	cmd  *exec.Cmd
	tty  *os.File
	resp chan *tmuxResponseT
	win  map[string]*WindowT
	pane map[string]*PaneT
	keys keyBindsT

	activeWindow *WindowT
	renderer     types.Renderer

	limiter sync.Mutex
}

type tmuxResponseT struct {
	Message [][]byte
	IsErr   bool
}

const (
	START_ATTACH_SESSION = "attach-session"
	START_NEW_SESSION    = "new-session"
)

func NewStartSession(renderer types.Renderer, size *types.XY, startCommand string) (*Tmux, error) {
	tmux := &Tmux{
		resp:     make(chan *tmuxResponseT),
		win:      make(map[string]*WindowT),
		pane:     make(map[string]*PaneT),
		renderer: renderer,
	}

	var err error
	var allowExit bool
	resp := new(tmuxResponseT)

	tmux.cmd = exec.Command("tmux", "-CC", startCommand)
	tmux.cmd.Env = config.SetEnv()
	tmux.tty, err = pty.Start(tmux.cmd)
	if err != nil {
		return nil, err
	}

	// Discard the following because it's just setting mode:
	//    \u001bP1000p
	_, _ = tmux.tty.Read(make([]byte, 7))

	go func() {
		scanner := bufio.NewScanner(tmux.tty)

		for scanner.Scan() {
			b := scanner.Bytes()
			debug.Log(string(b))
			switch {
			case bytes.HasPrefix(b, _RESP_OUTPUT):
				params := bytes.SplitN(b, []byte{' '}, 3)
				paneId := string(params[1])
				pane, ok := tmux.pane[paneId]
				if ok {
					pane.buf.Write(octal.Unescape(params[2]))
					continue
				}
				msg := make([]byte, len(params[2]))
				copy(msg, params[2])
				go func() {
					err = tmux.updatePaneInfo(paneId)
					if err != nil {
						renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
						//return //continue
						panic("cannot create pane")
					}
					pane, ok = tmux.pane[paneId]
					if !ok {
						panic("cannot create pane")
					}
					pane.buf.Write(octal.Unescape(msg))
					go renderer.RefreshWindowList()
				}()

			case bytes.HasPrefix(b, _RESP_BEGIN):
				resp = new(tmuxResponseT)

			case bytes.HasPrefix(b, _RESP_ERROR):
				resp.IsErr = true
				fallthrough

			case bytes.HasPrefix(b, _RESP_END):
				tmux.resp <- resp

			case bytes.HasPrefix(b, _RESP_MESSAGE):
				msg := b[len(_RESP_MESSAGE):]
				renderer.DisplayNotification(types.NOTIFY_INFO, string(msg))

			case bytes.HasPrefix(b, _RESP_WINDOW_ADD):
				params := bytes.SplitN(b, []byte{' '}, 2)
				winId := string(params[1])
				_ = tmux.newWindow(winId)
				go func() {
					err := tmux.updatePaneInfo("")
					if err != nil {
						renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
					}
					err = tmux.updateWinInfo(winId)
					if err != nil {
						renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
					}
					err = tmux.SelectAndResizeWindow(winId, renderer.GetWindowSizeCells())
					if err != nil {
						renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
					}
					renderer.RefreshWindowList()
				}()

			case bytes.HasPrefix(b, _RESP_WINDOW_RENAMED):
				params := bytes.SplitN(b, []byte{' '}, 3)
				tmux.win[string(params[1])].Name = string(params[2])
				go renderer.RefreshWindowList()

			case bytes.HasPrefix(b, _RESP_WINDOW_CLOSE) || bytes.HasPrefix(b, _RESP_UNLINKED_WINDOW_CLOSE):
				params := bytes.SplitN(b, []byte{' '}, 3)
				win, ok := tmux.win[string(params[1])]
				if !ok {
					// window doesn't exist so lets not fret about it being closed
					continue
				}
				win.Close()

			case bytes.HasPrefix(b, _RESP_EXIT):
				if allowExit {
					exit.Exit(0)
				}

			default:
				// ignore anything that looks like a notification
				if ignoreResponse(b) {
					continue
				}

				message := make([]byte, len(b))
				copy(message, b)
				resp.Message = append(resp.Message, message)
			}
		}
	}()

	startMessage := <-tmux.resp
	if startMessage.IsErr {
		err := errors.New(string(bytes.Join(startMessage.Message, []byte(": "))))
		return nil, err
	}

	allowExit = true

	err = tmux.initSession(renderer, size)
	if err != nil {
		return nil, err
	}

	return tmux, nil // could shouldn't reach this point
}

func (tmux *Tmux) initSession(renderer types.Renderer, size *types.XY) error {
	err := tmux.RefreshClient(size)
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

func ignoreResponse(b []byte) bool {
	if len(b) > 0 && b[0] == '%' {
		for _, notification := range respIgnored {
			if bytes.HasPrefix(b, notification) {
				return true
			}
		}
	}

	return false
}

func (tmux *Tmux) SendCommand(b []byte) (*tmuxResponseT, error) {
	//debug.Log(fmt.Sprintf("waiting for lock (%s)", string(b)))
	tmux.limiter.Lock()
	//debug.Log(fmt.Sprintf("lock granted (%s)", string(b)))

	_, err := tmux.tty.Write(append(b, '\n'))
	if err != nil {
		debug.Log(fmt.Sprintf("error (%s): %v", string(b), err))
		return nil, err
	}

	//debug.Log(fmt.Sprintf("command sent (%s)", string(b)))

	resp := <-tmux.resp

	//debug.Log(fmt.Sprintf("response received (%s)", string(b)))

	tmux.limiter.Unlock()
	//debug.Log(fmt.Sprintf("unlocked (%s)", string(b)))

	if resp.IsErr {
		return nil, fmt.Errorf("tmux command failed: %s", string(bytes.Join(resp.Message, []byte(": "))))
	}

	return resp, nil
}

func (tmux *Tmux) NewWindow() {
	_, err := tmux.SendCommand([]byte("new-window"))
	if err != nil {
		tmux.renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
	}
}
