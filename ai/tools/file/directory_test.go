package filetools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/lmorg/ttyphoon/types"
)

type directoryTestAgent struct {
	pathTestAgent
	renderer types.Renderer
}

func (a *directoryTestAgent) Renderer() types.Renderer { return a.renderer }

type directoryTestRenderer struct{}

func (directoryTestRenderer) Start(*types.AppWindowTerms, any, context.Context) {}
func (directoryTestRenderer) ShowAndFocusWindow()                               {}
func (directoryTestRenderer) GetWindowSizeCells() *types.XY                     { return nil }
func (directoryTestRenderer) GetGlyphSize() *types.XY                           { return nil }
func (directoryTestRenderer) GetBlinkState() bool                               { return false }
func (directoryTestRenderer) SetBlinkState(bool)                                {}
func (directoryTestRenderer) PrintCell(types.Tile, *types.Cell, *types.XY)      {}
func (directoryTestRenderer) PrintRow(types.Tile, []*types.Cell, *types.XY)     {}
func (directoryTestRenderer) DrawFrame(types.Tile)                              {}
func (directoryTestRenderer) DrawGaugeH(types.Tile, *types.XY, int32, int, int, *types.Colour) {
}
func (directoryTestRenderer) DrawGaugeV(types.Tile, *types.XY, int32, int, int, *types.Colour) {
}
func (directoryTestRenderer) DrawTable(types.Tile, *types.XY, int32, []int32) {}
func (directoryTestRenderer) DrawHighlightRect(types.Tile, *types.XY, *types.XY) {
}
func (directoryTestRenderer) DrawRectWithColour(types.Tile, *types.XY, *types.XY, *types.Colour, bool) {
}
func (directoryTestRenderer) DrawRectWithColourAndBorder(types.Tile, *types.XY, *types.XY, *types.Colour, bool, bool) {
}
func (directoryTestRenderer) DrawOutputBlockChrome(types.Tile, int32, int32, *types.Colour, bool) {
}
func (directoryTestRenderer) GetWindowTitle() string { return "" }
func (directoryTestRenderer) SetWindowTitle(string)  {}
func (directoryTestRenderer) StatusBarText(string)   {}
func (directoryTestRenderer) RefreshWindowList()     {}
func (directoryTestRenderer) Bell()                  {}
func (directoryTestRenderer) TriggerRedraw()         {}
func (directoryTestRenderer) TriggerLazyRedraw()     {}
func (directoryTestRenderer) TriggerDeallocation(func()) {
}
func (directoryTestRenderer) TriggerQuit() {}
func (directoryTestRenderer) NewElement(types.Tile, types.ElementID, ...any) types.Element {
	return nil
}
func (directoryTestRenderer) DisplayNotification(types.NotificationType, string) {}
func (directoryTestRenderer) DisplaySticky(types.NotificationType, string, func()) types.Notification {
	return nil
}
func (directoryTestRenderer) DisplayInputBox(string, string, types.InputBoxCallbackT, types.InputBoxCallbackT) {
}
func (directoryTestRenderer) DisplayInputBoxW(*types.InputBoxWT) {}
func (directoryTestRenderer) DisplayMenu(string, []string, types.MenuCallbackT, types.MenuCallbackT, types.MenuCallbackT) {
}
func (directoryTestRenderer) NewContextMenu() types.ContextMenu           { return nil }
func (directoryTestRenderer) AddToContextMenu(...types.MenuItem)          {}
func (directoryTestRenderer) DisplayMarkdownModel(string)                 {}
func (directoryTestRenderer) ShowTooltip(string)                          {}
func (directoryTestRenderer) CloseTooltip()                               {}
func (directoryTestRenderer) ResizeWindow(*types.XY)                      {}
func (directoryTestRenderer) SetKeyboardFnMode(types.KeyboardMode)        {}
func (directoryTestRenderer) GetKeyboardModifier() int                    { return 0 }
func (directoryTestRenderer) RefreshNotes()                               {}
func (directoryTestRenderer) NotesEditFile(string)                        {}
func (directoryTestRenderer) NotesCreateAndOpen(string, string)           {}
func (directoryTestRenderer) DisplayImageFullscreen(string, int32, int32) {}
func (directoryTestRenderer) ActiveTile() types.Tile                      { return nil }
func (directoryTestRenderer) GetWindowContext() context.Context           { return context.Background() }
func (directoryTestRenderer) AskAi()                                      {}
func (directoryTestRenderer) Close()                                      {}

func TestDirectoryCallReturnsFilesInProjectRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "alpha.txt"), []byte("alpha"), 0644); err != nil {
		t.Fatalf("WriteFile(alpha): %v", err)
	}
	if err := os.Mkdir(filepath.Join(root, "nested"), 0755); err != nil {
		t.Fatalf("Mkdir(nested): %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "nested", "beta.txt"), []byte("beta"), 0644); err != nil {
		t.Fatalf("WriteFile(beta): %v", err)
	}

	tool := &Directory{
		agent: &directoryTestAgent{
			pathTestAgent: pathTestAgent{projectRoot: root},
			renderer:      directoryTestRenderer{},
		},
	}

	response, err := tool.Call(context.Background(), "")
	if err != nil {
		t.Fatalf("Call() error = %v", err)
	}

	var got []file
	if err := json.Unmarshal([]byte(response), &got); err != nil {
		t.Fatalf("Unmarshal(%q): %v", response, err)
	}

	seen := map[string]bool{}
	for _, item := range got {
		seen[item.Name] = item.IsDir
	}
	if _, ok := seen["alpha.txt"]; !ok {
		t.Fatalf("directory response missing alpha.txt: %q", response)
	}
	if isDir, ok := seen["nested"]; !ok || !isDir {
		t.Fatalf("directory response missing nested directory: %q", response)
	}
	if _, ok := seen[filepath.Join("nested", "beta.txt")]; !ok {
		t.Fatalf("directory response missing nested/beta.txt: %q", response)
	}
}
