package rendererwebkit

import (
	"testing"

	"github.com/lmorg/ttyphoon/types"
)

type paneLayoutTestTile struct {
	types.Tile
	id   string
	x, y int32
	term types.Term
}

func (tile *paneLayoutTestTile) Id() string          { return tile.id }
func (tile *paneLayoutTestTile) Name() string        { return tile.id }
func (tile *paneLayoutTestTile) Left() int32         { return tile.x }
func (tile *paneLayoutTestTile) Right() int32        { return tile.x + tile.term.GetSize().X - 1 }
func (tile *paneLayoutTestTile) Top() int32          { return tile.y }
func (tile *paneLayoutTestTile) Bottom() int32       { return tile.y + tile.term.GetSize().Y - 1 }
func (tile *paneLayoutTestTile) GetTerm() types.Term { return tile.term }

type completionFrameTestTile struct {
	paneLayoutTestTile
	colour *types.Colour
}

func (tile *completionFrameTestTile) Right() int32                           { return tile.x + tile.term.GetSize().X - 1 }
func (tile *completionFrameTestTile) Bottom() int32                          { return tile.y + tile.term.GetSize().Y - 1 }
func (tile *completionFrameTestTile) InteractiveActionChrome() *types.Colour { return tile.colour }

func TestCompletedPaneFrameCarriesBackgroundResult(t *testing.T) {
	for _, test := range []struct {
		colour *types.Colour
		flags  int32
	}{{nil, 0}, {types.COLOR_OK, 1}, {types.COLOR_ERROR, 2}} {
		renderer := &webkitRender{}
		tile := &completionFrameTestTile{
			paneLayoutTestTile: paneLayoutTestTile{term: &paneLayoutTestTerm{size: types.XY{X: 80, Y: 24}}},
			colour:             test.colour,
		}
		renderer.DrawFrame(tile)
		if len(renderer.drawCommands) != 1 || renderer.drawCommands[0].Op != DrawOpFrame || renderer.drawCommands[0].Flags != test.flags {
			t.Fatalf("completion frame = %+v, want frame with result %d", renderer.drawCommands, test.flags)
		}
		ops := encodeDrawCommands(renderer.drawCommands)
		if len(ops) != 1 || len(ops[0]) != 6 || ops[0][5] != test.flags {
			t.Fatalf("compact completion frame = %v, want result %d", ops, test.flags)
		}
	}
}

func TestStackedPaneFramesSplitDividerRowAtHalfHeight(t *testing.T) {
	upper := &paneLayoutTestTile{id: "%upper", term: &paneLayoutTestTerm{size: types.XY{X: 80, Y: 23}}}
	lower := &paneLayoutTestTile{id: "%lower", y: 24, term: &paneLayoutTestTerm{size: types.XY{X: 80, Y: 24}}}
	side := &paneLayoutTestTile{id: "%side", x: 82, term: &paneLayoutTestTerm{size: types.XY{X: 80, Y: 23}}}
	renderer := &webkitRender{termWin: &types.AppWindowTerms{Tiles: []types.Tile{upper, lower, side}}}

	renderer.DrawFrame(upper)
	if got := renderer.drawCommands[0].Flags; got != frameFlagBottomHalf {
		t.Fatalf("upper pane frame flags = %b, want bottom half-row inset", got)
	}
	renderer.drawCommands = nil
	renderer.DrawFrame(lower)
	if got := renderer.drawCommands[0].Flags; got != frameFlagTopHalf {
		t.Fatalf("lower pane frame flags = %b, want top half-row inset", got)
	}
	renderer.drawCommands = nil
	renderer.DrawFrame(side)
	if got := renderer.drawCommands[0].Flags; got != 0 {
		t.Fatalf("side-by-side pane frame flags = %b, want no divider inset", got)
	}
}

type paneLayoutTestTerm struct {
	types.Term
	size    types.XY
	focused bool
}

func (term *paneLayoutTestTerm) GetSize() *types.XY { return &term.size }
func (term *paneLayoutTestTerm) Render() bool       { return true }
func (term *paneLayoutTestTerm) IsFocused() bool    { return term.focused }

func TestInactivePaneOverlayCoversItsHalfOfStackedDivider(t *testing.T) {
	upper := &paneLayoutTestTile{id: "%upper", term: &paneLayoutTestTerm{size: types.XY{X: 80, Y: 23}}}
	lower := &paneLayoutTestTile{id: "%lower", y: 24, term: &paneLayoutTestTerm{size: types.XY{X: 80, Y: 24}, focused: true}}
	renderer := &webkitRender{termWin: &types.AppWindowTerms{Tiles: []types.Tile{upper, lower}}}

	commands := renderer.PopDrawCommands()
	var dividerOverlay *DrawCommand
	for i := range commands {
		if commands[i].Op == DrawOpTileOverlay && commands[i].Flags == tileOverlayTopHalf {
			dividerOverlay = &commands[i]
			break
		}
	}
	if dividerOverlay == nil {
		t.Fatal("inactive upper pane has no divider half-overlay")
	}
	if dividerOverlay.X != upper.Left() || dividerOverlay.Y != upper.Bottom()+1 || dividerOverlay.Height != 1 {
		t.Fatalf("divider overlay geometry = %+v", *dividerOverlay)
	}
	if len(encodeDrawCommands([]DrawCommand{*dividerOverlay})[0]) != 7 {
		t.Fatal("compact divider overlay omitted its half-row flag")
	}
}

func TestRefreshWindowListBeforeTmuxInitializationIsSafe(t *testing.T) {
	(&webkitRender{}).RefreshWindowList()
}

func TestSplitRowsDoNotDoubleDimAdjacentDividerEdges(t *testing.T) {
	middle := &paneLayoutTestTile{id: "%middle", y: 8, term: &paneLayoutTestTerm{size: types.XY{X: 81, Y: 7}}}
	tiles := []types.Tile{middle}
	for _, row := range []int32{0, 16} {
		for _, column := range []int32{0, 41} {
			tiles = append(tiles, &paneLayoutTestTile{
				x: column, y: row,
				term: &paneLayoutTestTerm{size: types.XY{X: 40, Y: 7}, focused: true},
			})
		}
	}
	renderer := &webkitRender{termWin: &types.AppWindowTerms{Tiles: tiles}}
	commands := renderer.PopDrawCommands()
	counts := map[int32]int{}
	for _, command := range commands {
		if command.Op == DrawOpTileOverlay && command.Flags != 0 {
			counts[command.Y]++
		}
	}
	if len(counts) != 2 || counts[7] != 1 || counts[15] != 1 {
		t.Fatalf("divider overlay counts = %v; want one strip per edge", counts)
	}
}

func TestPaneLayoutTracksVisiblePaneGeometry(t *testing.T) {
	first := &paneLayoutTestTile{id: "%1", term: &paneLayoutTestTerm{size: types.XY{X: 80, Y: 24}}}
	secondTerm := &paneLayoutTestTerm{size: types.XY{X: 80, Y: 24}}
	second := &paneLayoutTestTile{id: "%2", x: 81, y: 3, term: secondTerm}
	renderer := &webkitRender{termWin: &types.AppWindowTerms{Tiles: []types.Tile{second, nil, first}}}
	panes := renderer.GetPaneLayout()
	if len(panes) != 2 || panes[0].ID != "%1" || panes[1].ID != "%2" {
		t.Fatalf("unexpected pane order: %+v", panes)
	}
	if panes[1].X != 81 || panes[1].Y != 3 || panes[1].Width != 80 || panes[1].Height != 24 || panes[1].CanClose {
		t.Fatalf("unexpected pane geometry: %+v", panes[1])
	}
	second.x = 0
	second.y = 0
	secondTerm.size = types.XY{X: 120, Y: 40}
	renderer.termWin.Tiles = []types.Tile{second}
	panes = renderer.GetPaneLayout()
	if len(panes) != 1 || panes[0].ID != "%2" || panes[0].X != 0 || panes[0].Y != 0 || panes[0].Width != 120 || panes[0].Height != 40 {
		t.Fatalf("stale pane layout: %+v", panes)
	}
	renderer.termWin.Tiles = nil
	if len(renderer.GetPaneLayout()) != 0 {
		t.Fatal("removed panes remain in layout")
	}
}
