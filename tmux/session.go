package tmux

import (
	"fmt"
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
