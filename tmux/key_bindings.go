package tmux

import (
	"strings"

	"github.com/lmorg/mxtty/codes"
	"github.com/lmorg/mxtty/hotkeys"
)

type fnKeyT func(*Tmux) error

/*
	tmux list-keys -Na:
	C-b C-o     Rotate through the panes
	C-b C-z     Suspend the current client
	C-b Space   Select next layout
	C-b !       Break pane to a new window
	C-b "       Split window vertically
	C-b #       List all paste buffers
	C-b $       Rename current session
	C-b &       Kill current window
	C-b '       Prompt for window index to select
	C-b (       Switch to previous client
	C-b )       Switch to next client
	C-b ,       Rename current window
	C-b .       Move the current window
	C-b /       Describe key binding
	C-b 0       Select window 0
	C-b 1       Select window 1
	C-b 2       Select window 2
	C-b 3       Select window 3
	C-b 4       Select window 4
	C-b 5       Select window 5
	C-b 6       Select window 6
	C-b 7       Select window 7
	C-b 8       Select window 8
	C-b 9       Select window 9
	C-b :       Prompt for a command
	C-b ;       Move to the previously active pane
	C-b ?       List key bindings
	C-b C       Customize options
	C-b D       Choose and detach a client from a list
	C-b E       Spread panes out evenly
	C-b L       Switch to the last client
	C-b ]       Paste the most recent paste buffer
	C-b c       Create a new window
	C-b d       Detach the current client
	C-b f       Search for a pane
	C-b i       Display window information
	C-b l       Select the previously current window
	C-b n       Select the next window
	C-b o       Select the next pane
	C-b q       Display pane numbers
	C-b s       Choose a session from a list
	C-b t       Show a clock
	C-b w       Choose a window from a list
	C-b x       Kill the active pane
	C-b z       Zoom the active pane
	C-b {       Swap the active pane with the pane above
	C-b }       Swap the active pane with the pane below
	C-b ~       Show messages
	C-b DC      Reset so the visible part of the window follows the cursor
	C-b PPage   Enter copy mode and scroll up
	C-b Up      Select the pane above the active pane
	C-b Down    Select the pane below the active pane
	C-b Left    Select the pane to the left of the active pane
	C-b Right   Select the pane to the right of the active pane
	C-b M-1     Set the even-horizontal layout
	C-b M-2     Set the even-vertical layout
	C-b M-3     Set the main-horizontal layout
	C-b M-4     Set the main-vertical layout
	C-b M-5     Select the tiled layout
	C-b M-n     Select the next window with an alert
	C-b M-o     Rotate through the panes in reverse
	C-b M-p     Select the previous window with an alert
	C-b M-Up    Resize the pane up by 5
	C-b M-Down  Resize the pane down by 5
	C-b M-Left  Resize the pane left by 5
	C-b M-Right Resize the pane right by 5
	C-b C-Up    Resize the pane up
	C-b C-Down  Resize the pane down
	C-b C-Left  Resize the pane left
	C-b C-Right Resize the pane right
	C-b S-Up    Move the visible part of the window up
	C-b S-Down  Move the visible part of the window down
	C-b S-Left  Move the visible part of the window left
	C-b S-Right Move the visible part of the window right
*/

var defaultFnKeys = map[string]fnKeyT{
	"Create a new window":                             fnKeyNewWindow,
	"Kill the active pane":                            fnKeyKillPane,
	"Kill current window":                             fnKeyKillCurrentWindow,
	"Choose a window from a list":                     fnKeyChooseWindowFromList,
	"Rename current window":                           fnKeyRenameWindow,
	"Select window 0":                                 fnKeySelectWindow0,
	"Select window 1":                                 fnKeySelectWindow1,
	"Select window 2":                                 fnKeySelectWindow2,
	"Select window 3":                                 fnKeySelectWindow3,
	"Select window 4":                                 fnKeySelectWindow4,
	"Select window 5":                                 fnKeySelectWindow5,
	"Select window 6":                                 fnKeySelectWindow6,
	"Select window 7":                                 fnKeySelectWindow7,
	"Select window 8":                                 fnKeySelectWindow8,
	"Select window 9":                                 fnKeySelectWindow9,
	"Move to the previously active pane":              fnKeyLastPane,
	"Select the previously current window":            fnKeyLastWindow,
	"Select the next window with an alert":            fnKeyNextWindowAlert,
	"Select the previous window with an alert":        fnKeyPreviousWindowAlert,
	"List key bindings":                               fnKeyListBindings,
	"Split window horizontally":                       fnKeySplitWindowHorizontally,
	"split-window -h":                                 fnKeySplitWindowHorizontally,
	"Split window vertically":                         fnKeySplitWindowVertically,
	"split-window -v":                                 fnKeySplitWindowVertically,
	"Select the pane above the active pane":           fnKeySelectPaneUp,
	"Select the pane below the active pane":           fnKeySelectPaneDown,
	"Select the pane to the left of the active pane":  fnKeySelectPaneLeft,
	"Select the pane to the right of the active pane": fnKeySelectPaneRight,
	"Spread panes out evenly":                         fnKeyTilePanes,
	"Resize the pane up by 5":                         fnKeyResizePaneUp5,
	"Resize the pane down by 5":                       fnKeyResizePaneDown5,
	"Resize the pane left by 5":                       fnKeyResizePaneLeft5,
	"Resize the pane right by 5":                      fnKeyResizePaneRight5,
	"Resize the pane up":                              fnKeyResizePaneUp1,
	"Resize the pane down":                            fnKeyResizePaneDown1,
	"Resize the pane left":                            fnKeyResizePaneLeft1,
	"Resize the pane right":                           fnKeyResizePaneRight1,
}

func (tmux *Tmux) _getDefaultTmuxKeyBindings() error {
	const (
		PREFIX = iota
		KEY
		NOTE
	)

	resp, err := tmux.SendCommand([]byte(`list-keys -N -a`))
	if err != nil {
		return err
	}

	for i := range resp.Message {
		split := strings.SplitN(string(resp.Message[i]), " ", 3)
		note := strings.TrimSpace(split[NOTE])
		if fn, ok := defaultFnKeys[note]; ok {

			hotkeys.Add(
				codes.KeyName(split[PREFIX]),
				codes.KeyName(split[KEY]),
				func() error { return fn(tmux) },
				note)
		}
	}

	return nil
}
