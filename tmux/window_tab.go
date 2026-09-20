package tmux

import (
	"fmt"
	"sort"

	"github.com/lmorg/ttyphoon/debug"
	"github.com/lmorg/ttyphoon/types"
)

func (tmux *Tmux) GetTermTiles() *types.AppWindowTerms {
	_ = tmux.updateWinInfo("")
	_ = tmux.updatePaneInfo("")

	aw := new(types.AppWindowTerms)

	var zoomed *PaneT
	for pane := range tmux.activeWindow.panes.Each() {
		if pane.closed {
			debug.Log(fmt.Sprintf("skipping closed pane %s", pane.id))
			pane.exit()
			continue
		}
		if pane.zoomed {
			zoomed = pane
		}
		aw.Tiles = append(aw.Tiles, pane)
	}

	// When a pane is zoomed tmux maximises it to fill the window and hides the
	// others. Mirror that by rendering only the zoomed pane.
	if zoomed != nil {
		aw.Tiles = []types.Tile{zoomed}
	}

	aw.Active = tmux.activeWindow.ActivePane()

	for win := range tmux.wins.Each() {
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

func (win *WindowT) Name() string {
	if win != nil {
		return win.name
	}
	return ""
}

func (win *WindowT) Id() string   { return win.id }
func (win *WindowT) Index() int   { return win.index }
func (win *WindowT) Active() bool { return win.active }
func (win *WindowT) Rename(name string) error {
	return win.activePane.tmux.RenameWindow(win, name)
}
