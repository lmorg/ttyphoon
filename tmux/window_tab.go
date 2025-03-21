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

	//tmux.activeWindow.panes.mutex.Lock()
	for pane := range tmux.activeWindow.panes.Each() {
		if pane.closed {
			debug.Log(fmt.Sprintf("skipping closed pane %s", pane.id))
			pane.exit()
			continue
		}
		aw.Tiles = append(aw.Tiles, pane)
	}
	//tmux.activeWindow.panes.mutex.Unlock()

	aw.Active = tmux.activeWindow.ActivePane()

	for _, win := range tmux.wins {
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
func (win *WindowT) Rename(name string) error {
	return win.activePane.tmux.RenameWindow(win, name)
}
