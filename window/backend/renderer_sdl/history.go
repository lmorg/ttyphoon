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

func Open(renderer *sdlRender, tile types.Tile) {
	path := fmt.Sprintf("%s/Documents/%s/history/%s/", xdg.Home, app.DirName, tile.GroupName())
	entries, err := os.ReadDir(path)
	if err != nil {
		renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
		return
	}

	var files []string

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			files = append(files, entry.Name())
		}
	}

	if len(files) == 0 {
		renderer.DisplayNotification(types.NOTIFY_WARN, "There is no history for this group")
		return
	}

	sort.Sort(sort.Reverse(sort.StringSlice(files)))

	// start webview
	windowStyle := dispatcher.NewWindowStyle()
	windowStyle.Pos = types.XY{}
	x, y := renderer.window.GetSize()
	windowStyle.Size = types.XY{X: x, Y: y}
	windowStyle.Title = string(files[0])

	parameters := &dispatcher.PMarkdownT{Path: path + files[0]}

	ipc, closer := dispatcher.DisplayWindow("history", windowStyle, parameters, func(msg *dispatcher.IpcMessageT) {
		if msg.Error != nil {
			renderer.DisplayNotification(types.NOTIFY_ERROR, msg.Error.Error())
		} else {
			switch msg.EventName {
			case "focus":
				renderer.TriggerDeallocation(renderer.window.Raise)
			case "closeMenu":
				renderer.closeMenu()
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
			renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
		}
	}

	cancelFn := func(_ int) { closer() }

	renderer.DisplayMenu("History item to view", files, selectFn, nil, cancelFn)
}
