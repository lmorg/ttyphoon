package rendererwebkit

import (
	"testing"

	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/utils/notes"
)

type notesContextTestTile struct {
	types.Tile
	pwd   string
	group string
}

func (tile *notesContextTestTile) Pwd() string       { return tile.pwd }
func (tile *notesContextTestTile) GroupName() string { return tile.group }

type notesContextTestApp struct {
	updates []notes.FileContext
}

func (app *notesContextTestApp) SyncNotesContext(roots notes.FileContext) {
	app.updates = append(app.updates, roots)
}

func TestNotesFileContextReusesResolutionAndPublishesOnlyChanges(t *testing.T) {
	tile := &notesContextTestTile{pwd: t.TempDir(), group: "Workspace"}
	roots := notes.FileContext{
		PathProjectRoot: tile.pwd,
		PathGlobalNotes: "/global/",
		PathUserNotes:   "/global/Workspace/",
		GroupName:       tile.group,
	}
	app := &notesContextTestApp{}
	renderer := &webkitRender{
		termWin:       &types.AppWindowTerms{Active: tile},
		app:           app,
		notesResolved: roots,
		notesPwd:      tile.pwd,
		notesGroup:    tile.group,
	}
	for range 3 {
		if got := renderer.NotesFileContext(); got != roots {
			t.Fatalf("context = %+v, want cached %+v", got, roots)
		}
	}
	if len(app.updates) != 1 {
		t.Fatalf("context updates = %d, want 1", len(app.updates))
	}
	newRoots := roots
	newRoots.GroupName = "Other"
	renderer.notesResolved = newRoots
	renderer.NotesFileContext()
	if len(app.updates) != 2 || app.updates[1] != newRoots {
		t.Fatalf("changed context was not published: %+v", app.updates)
	}
}
