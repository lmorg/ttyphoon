package rendersdl

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/adrg/xdg"
	"github.com/lmorg/ttyphoon/app"
	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/utils/dispatcher"
)

func (sr *sdlRender) OpenHistory(tile types.Tile) {
	path := fmt.Sprintf("%s/Documents/%s/history/%s/", xdg.Home, app.DirName, tile.GroupName())
	entries, err := os.ReadDir(path)
	if err != nil {
		sr.DisplayNotification(types.NOTIFY_ERROR, err.Error())
		return
	}

	var files []string

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			files = append(files, entry.Name())
		}
	}

	if len(files) == 0 {
		sr.DisplayNotification(types.NOTIFY_WARN, "There is no history for this group")
		return
	}

	sort.Sort(sort.Reverse(sort.StringSlice(files)))

	// start webview
	windowStyle := dispatcher.NewWindowStyle()
	windowStyle.Pos = types.XY{}
	x, y := sr.window.GetSize()
	windowStyle.Size = types.XY{X: x, Y: y}
	windowStyle.Title = string(files[0])

	parameters := &dispatcher.PMarkdownT{Path: path + files[0]}

	ipc, closer := dispatcher.DisplayWindow("history", windowStyle, parameters, func(msg *dispatcher.IpcMessageT) {
		if msg.Error != nil {
			sr.DisplayNotification(types.NOTIFY_ERROR, msg.Error.Error())
		} else {
			switch msg.EventName {
			case "focus":
				sr.TriggerDeallocation(sr.window.Raise)
			case "closeMenu":
				sr.closeMenu()
			}
		}
	})

	selectFn := func(i int) {
		err := ipc.Send(&dispatcher.IpcMessageT{
			EventName: "markdownOpen",
			Parameters: map[string]string{
				"path": path + files[i],
			},
		})
		if err != nil {
			sr.DisplayNotification(types.NOTIFY_ERROR, err.Error())
		}
	}

	cancelFn := func(_ int) { closer() }

	okFn := func(_ int) {
		err := ipc.Send(&dispatcher.IpcMessageT{
			EventName: "focus",
		})
		if err != nil {
			sr.DisplayNotification(types.NOTIFY_ERROR, err.Error())
		}
	}

	sr.DisplayMenu("History item to view", files, selectFn, okFn, cancelFn)
}
