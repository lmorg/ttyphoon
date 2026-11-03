package rendererwebkit

import (
	"testing"

	"github.com/lmorg/ttyphoon/types"
)

type mouseLeaveTestTerm struct {
	types.Term
	leaveCalls int
	hoverCalls int
}

func (term *mouseLeaveTestTerm) MouseLeave()          { term.leaveCalls++ }
func (term *mouseLeaveTestTerm) MouseHover(*types.XY) { term.hoverCalls++ }
func (term *mouseLeaveTestTerm) GetSize() *types.XY   { return &types.XY{X: 80, Y: 24} }

type mouseLeaveTestTile struct {
	types.Tile
	term types.Term
}

func (tile *mouseLeaveTestTile) GetTerm() types.Term { return tile.term }
func (tile *mouseLeaveTestTile) Left() int32         { return 0 }
func (tile *mouseLeaveTestTile) Right() int32        { return 80 }
func (tile *mouseLeaveTestTile) Top() int32          { return 0 }
func (tile *mouseLeaveTestTile) Bottom() int32       { return 24 }

func TestMouseLeaveClearsAllPaneHoverAndDisablesReplay(t *testing.T) {
	first := &mouseLeaveTestTerm{}
	second := &mouseLeaveTestTerm{}
	tile := &mouseLeaveTestTile{term: first}
	renderer := &webkitRender{
		termWin: &types.AppWindowTerms{
			Active: tile,
			Tiles:  []types.Tile{tile, &mouseLeaveTestTile{term: second}},
		},
		_redraw:       make(chan struct{}, 1),
		statusBarText: "[Click] Fold branch",
	}
	renderer.setLastMouseCell(2, 3)
	renderer.applyMouseHoverFromLastPosition()
	if first.hoverCalls != 1 {
		t.Fatal("hover did not run while the pointer was inside")
	}
	renderer.HandleMouseLeave()
	renderer.applyMouseHoverFromLastPosition()
	if renderer.lastMouseValid.Load() || first.hoverCalls != 1 {
		t.Fatal("hover replayed after the pointer left")
	}
	if first.leaveCalls != 1 || second.leaveCalls != 1 {
		t.Fatalf("pane leave calls = %d, %d, want 1, 1", first.leaveCalls, second.leaveCalls)
	}
	if renderer.statusBarText != "" || len(renderer._redraw) != 1 {
		t.Fatal("leave did not clear hover status and request a redraw")
	}
	if renderer.termWin.Active != tile {
		t.Fatal("leave changed the active pane")
	}
	renderer.setLastMouseCell(2, 3)
	renderer.applyMouseHoverFromLastPosition()
	if first.hoverCalls != 2 {
		t.Fatal("hover did not resume on re-entry")
	}
}
