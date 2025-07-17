package tmux

import (
	"bufio"
	"bytes"
	"errors"
	"os"
	"os/exec"
	"sync"
	"time"

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
	_RESP_OUTPUT = "%output"
	_RESP_BEGIN  = "%begin"
	_RESP_END    = "%end"
	_RESP_ERROR  = "%error"

	_RESP_CLIENT_DETACHED         = "%client-detached"
	_RESP_CLIENT_SESSION_CHANGED  = "%client-session-changed"
	_RESP_CONFIG_ERROR            = "%config-error"
	_RESP_CONTINUE                = "%continue"
	_RESP_EXIT                    = "%exit"
	_RESP_EXTENDED_OUTPUT         = "%extended-output"
	_RESP_LAYOUT_CHANGE           = "%layout-change"
	_RESP_MESSAGE                 = "%message"
	_RESP_PANE_MODE_CHANGED       = "%pane-mode-changed"
	_RESP_PASTE_BUFFER_CHANGED    = "%paste-buffer-changed"
	_RESP_PASTE_BUFFER_DELETED    = "%paste-buffer-deleted"
	_RESP_PAUSE                   = "%pause"
	_RESP_SESSION_CHANGED         = "%session-changed"
	_RESP_SESSION_RENAMED         = "%session-renamed"
	_RESP_SESSION_WINDOW_CHANGED  = "%session-window-changed"
	_RESP_SESSIONS_CHANGED        = "%sessions-changed"
	_RESP_SUBSCRIPTION_CHANGED    = "%subscription-changed"
	_RESP_UNLINKED_WINDOW_ADD     = "%unlinked-window-add"
	_RESP_UNLINKED_WINDOW_CLOSE   = "%unlinked-window-close"
	_RESP_UNLINKED_WINDOW_RENAMED = "%unlinked-window-renamed"
	_RESP_WINDOW_ADD              = "%window-add"
	_RESP_WINDOW_CLOSE            = "%window-close"
	_RESP_WINDOW_PANE_CHANGED     = "%window-pane-changed"
	_RESP_WINDOW_RENAMED          = "%window-renamed"
)

type Tmux struct {
	cmd   *exec.Cmd
	tty   *os.File
	resp  chan *tmuxResponseT
	_resp *tmuxResponseT
	wins  windowMap //map[string]*WindowT
	panes paneMap   //map[string]*PaneT

	keys      keyBindsT
	allowExit bool

	appWindow    *types.AppWindowTerms
	activeWindow *WindowT
	renderer     types.Renderer

	limiter   sync.Mutex
	prefixTtl time.Time
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
		wins:     newWindowMap(),
		panes:    newPaneMap(),
		renderer: renderer,
	}

	var err error
	tmux._resp = new(tmuxResponseT)

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
			debug.Log(b)

			prefix := bytes.SplitN(b, []byte{' '}, 2)

			fn, ok := tmuxCommandMap[string(prefix[0])]
			if ok {
				fn(tmux, b)
			} else {
				_respDefault(tmux, b)
			}
		}
	}()

	startMessage := <-tmux.resp
	if startMessage.IsErr {
		err := errors.New(string(bytes.Join(startMessage.Message, []byte(": "))))
		return nil, err
	}

	tmux.allowExit = true

	err = tmux.initSession(renderer, size)
	if err != nil {
		return nil, err
	}

	return tmux, nil
}

var tmuxCommandMap = map[string]func(*Tmux, []byte){
	_RESP_OUTPUT:  _respOutput,
	_RESP_BEGIN:   _respBegin,
	_RESP_END:     _respEnd,
	_RESP_ERROR:   _respError,
	_RESP_MESSAGE: _respMessage,

	_RESP_CLIENT_DETACHED:         __respIgnored,
	_RESP_CLIENT_SESSION_CHANGED:  __respIgnored,
	_RESP_CONFIG_ERROR:            _respConfigError,
	_RESP_CONTINUE:                __respIgnored,
	_RESP_EXIT:                    _respExit,
	_RESP_LAYOUT_CHANGE:           __respIgnored,
	_RESP_EXTENDED_OUTPUT:         _respExtendedOutput,
	_RESP_PANE_MODE_CHANGED:       __respIgnored,
	_RESP_PASTE_BUFFER_CHANGED:    __respIgnored,
	_RESP_PASTE_BUFFER_DELETED:    __respIgnored,
	_RESP_PAUSE:                   __respIgnored,
	_RESP_SESSION_CHANGED:         __respIgnored,
	_RESP_SESSION_RENAMED:         _respSessionRenamed,
	_RESP_SESSION_WINDOW_CHANGED:  _respSessionWindowChanged,
	_RESP_SESSIONS_CHANGED:        __respIgnored,
	_RESP_SUBSCRIPTION_CHANGED:    __respIgnored,
	_RESP_UNLINKED_WINDOW_ADD:     __respIgnored,
	_RESP_UNLINKED_WINDOW_CLOSE:   __respIgnored,
	_RESP_UNLINKED_WINDOW_RENAMED: __respIgnored,
	_RESP_WINDOW_ADD:              _respWindowAdd,
	_RESP_WINDOW_CLOSE:            _respWindowClose,
	_RESP_WINDOW_PANE_CHANGED:     _respWindowPaneChanged,
	_RESP_WINDOW_RENAMED:          _respWindowRenamed,
}

func _respOutput(tmux *Tmux, b []byte) {
	params := bytes.SplitN(b, []byte{' '}, 3)
	paneId := string(params[1])
	if pane := tmux.panes.Get(paneId); pane != nil {
		pane.buf.Write(octal.Unescape([]byte(params[2])))
		return
	}

	msg := make([]byte, len(params[2]))
	copy(msg, params[2])

	go func() {
		err := tmux.updatePaneInfo(paneId)
		if err != nil {
			tmux.renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
			return
		}
		pane := tmux.panes.Get(paneId)
		if pane == nil {
			tmux.renderer.DisplayNotification(types.NOTIFY_ERROR, "pane not found: "+paneId)
			return
		}
		pane.buf.Write(octal.Unescape(msg))
	}()
}

func _respExtendedOutput(tmux *Tmux, b []byte) {
	panic(_RESP_EXTENDED_OUTPUT)
}

func _respMessage(tmux *Tmux, b []byte) {
	msg := string(b[len(_RESP_MESSAGE)+1:])
	if msg == _PANE_EXITED {
		go func() { errToNotification(tmux.renderer, tmux.paneExited()) }()
		return
	}

	tmux.renderer.DisplayNotification(types.NOTIFY_INFO, msg)
}

func _respWindowAdd(tmux *Tmux, b []byte) {
	params := bytes.SplitN(b, []byte{' '}, 2)
	winId := string(params[1])
	go func() {
		tmux.newWindow(winId, types.CALLER__respWindowAdd)
		tmux.renderer.RefreshWindowList()
	}()
}

func _respWindowRenamed(tmux *Tmux, b []byte) {
	//tmux.renderer.DisplayNotification(types.NOTIFY_DEBUG, string(b))
	params := bytes.SplitN(b, []byte{' '}, 3)
	win := tmux.wins.Get(string(params[1]))
	if win == nil {
		debug.Log("No window to rename with Id: " + string(params[1]))
		return
	}

	win.name = string(params[2])
	go tmux.renderer.RefreshWindowList()
}

func _respWindowPaneChanged(tmux *Tmux, b []byte) {
	//tmux.renderer.DisplayNotification(types.NOTIFY_DEBUG, string(b))
	params := bytes.SplitN(b, []byte{' '}, 3)
	go func() {
		errToNotification(tmux.renderer, tmux.updatePaneInfo(string(params[2])))
		tmux.renderer.RefreshWindowList()
	}()
}

func _respWindowClose(tmux *Tmux, b []byte) {
	params := bytes.SplitN(b, []byte{' '}, 3)
	go func() {
		tmux.CloseWindow(string(params[2]))
		tmux.renderer.RefreshWindowList()
	}()
}

func _respExit(tmux *Tmux, b []byte) {
	if tmux.allowExit {
		exit.Exit(0)
	}
}

func _respBegin(tmux *Tmux, b []byte) {
	tmux._resp = new(tmuxResponseT)
}

func _respEnd(tmux *Tmux, b []byte) {
	tmux.resp <- tmux._resp
}

func _respError(tmux *Tmux, b []byte) {
	tmux._resp.IsErr = true
	_respEnd(tmux, b)
}

func _respConfigError(tmux *Tmux, b []byte) {
	tmux.renderer.DisplayNotification(types.NOTIFY_ERROR, string(b))
}

func _respSessionRenamed(tmux *Tmux, b []byte) {
	//tmux.renderer.DisplayNotification(types.NOTIFY_DEBUG, string(b))
	_respSessionWindowChanged(tmux, b)
}

func _respSessionWindowChanged(tmux *Tmux, b []byte) {
	params := bytes.SplitN(b, []byte{' '}, 3)
	go func() {
		tmux.updateWinInfo(string(params[2]))
		tmux.renderer.RefreshWindowList()
	}()
}

func _respDefault(tmux *Tmux, b []byte) {
	message := make([]byte, len(b))
	copy(message, b)
	tmux._resp.Message = append(tmux._resp.Message, message)
	//tmux._resp.Message = append(tmux._resp.Message, b)
}

func __respIgnored(tmux *Tmux, b []byte) {
	// do nothing
	debug.Log(b)
}

func __respPanic(tmux *Tmux, b []byte) {
	// this is used for debugging
	panic(string(b))
}

func errToNotification(renderer types.Renderer, err error) {
	if err != nil {
		renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
	}
}
