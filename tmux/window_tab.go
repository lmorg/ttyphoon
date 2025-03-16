package tmux

import (
	"fmt"
	"sort"

	"github.com/lmorg/mxtty/debug"
	"github.com/lmorg/mxtty/types"
)

func (tmux *Tmux) GetTermTiles() *types.AppWindowTerms {
	_ = tmux.updateWinInfo("")
	_ = tmux.updatePaneInfo("")

	aw := new(types.AppWindowTerms)

	for _, pane := range tmux.activeWindow.panes {
		if pane.closed {
			debug.Log(fmt.Sprintf("skipping closed pane %s", pane.id))
			go pane.exit()
			continue
		}
		aw.Tiles = append(aw.Tiles, pane)
	}

	aw.Active = tmux.activeWindow.ActivePane()

	for _, win := range tmux.win {
		if win.closed {
			win.close(tmux)
			continue
		}
		aw.Tabs = append(aw.Tabs, win)
	}

	sort.Slice(aw.Tabs, func(i, j int) bool {
		return aw.Tabs[i].Index() < aw.Tabs[j].Index()
	})

	debug.Log(aw)

	tmux.appWindow = aw
	return aw
}

func (win *WindowT) Name() string { return win.name }
func (win *WindowT) Id() string   { return win.id }
func (win *WindowT) Index() int   { return win.index }
func (win *WindowT) Active() bool { return win.active }
