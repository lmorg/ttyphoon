package tmux

import (
	"fmt"

	"github.com/lmorg/mxtty/types"
)

/*
	client_activity         Time client last had activity
	client_cell_height      Height of each client cell in pixels
	client_cell_width       Width of each client cell in pixels
	client_control_mode     1 if client is in control mode
	client_created          Time client created
	client_discarded        Bytes discarded when client behind
	client_flags            List of client flags
	client_height           Height of client
	client_key_table        Current key table
	client_last_session     Name of the client's last session
	client_name             Name of client
	client_pid              PID of client process
	client_prefix           1 if prefix key has been pressed
	client_readonly         1 if client is read-only
	client_session          Name of the client's session
	client_termfeatures     Terminal features of client, if any
	client_termname         Terminal name of client
	client_termtype         Terminal type of client, if available
	client_tty              Pseudo terminal of client
	client_uid              UID of client process
	client_user             User of client process
	client_utf8             1 if client supports UTF-8
	client_width            Width of client
	client_written          Bytes written to client
*/

var CMD_CLIENT_REFRESH = "refresh-client"

type clientT struct {
	Name         string `tmux:"client_name"`
	SessionName  string `tmux:"client_session"`
	ControlMode  bool   `tmux:"?client_control_mode,true,false"`
	Width        int    `tmux:"client_width"`
	Height       int    `tmux:"client_height"`
	TermFeatures string `tmux:"client_termfeatures"`
	TermName     string `tmux:"client_termname"`
	TermType     string `tmux:"client_termtype"`
	Tty          string `tmux:"client_tty"`
	Utf8         bool   `tmux:"?client_utf8,true,false"`
}

func (tmux *Tmux) RefreshClient(size *types.XY) error {
	/*strSize := fmt.Sprintf("%dx%d", size.X, size.Y)
	_, err := tmux.SendCommandWithReflection(CMD_CLIENT_REFRESH, reflect.TypeOf(clientT{}), "-C", strSize)
	if err != nil {
		return err
	}*/
	strSize := fmt.Sprintf("%s -C %dx%d", CMD_CLIENT_REFRESH, size.X, size.Y)
	_, err := tmux.SendCommand([]byte(strSize))
	if err != nil {
		return err
	}

	return nil
}
