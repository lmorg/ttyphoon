package rendersdl

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/utils/dispatcher"
)

var notes struct {
	ipc   *dispatcher.IpcT
	mutex sync.Mutex
}

func (sr *sdlRender) startNotes(tile types.Tile) {
	if !notes.mutex.TryLock() {
		// only run once
		return
	}
	defer notes.mutex.Unlock()
	if notes.ipc != nil {
		return
	}

	windowStyle := dispatcher.NewWindowStyle()
	windowStyle.Pos = types.XY{}
	x, y := sr.window.GetSize()
	windowStyle.Size = types.XY{X: x, Y: y}
	windowStyle.Title = "Notes"

	parameters := &dispatcher.PNotesT{
		ProjectRoot: findProjectRoot(tile.Pwd()),
		UserNotes:   userDocs(tile, "notes"),
	}

	notes.ipc, _ = dispatcher.DisplayWindow(dispatcher.WindowNotes, windowStyle, parameters, func(msg *dispatcher.IpcMessageT) {
		if msg.Error != nil {
			sr.DisplayNotification(types.NOTIFY_ERROR, msg.Error.Error())
		} else {
			switch msg.EventName {
			case "focus":
				sr.TriggerDeallocation(sr.window.Raise)
			}
		}
	})
}

func (sr *sdlRender) openNotes() {
	tile := sr.termWin.Active
	sr.startNotes(tile)

	//sr.UpdateNotes(tile)

	err := notes.ipc.Send(&dispatcher.IpcMessageT{
		EventName: "notesFocus",
		Parameters: map[string]string{
			"projectRoot": findProjectRoot(tile.Pwd()),
			"userNotes":   userDocs(tile, "notes"),
		},
	})
	if err != nil {
		sr.DisplayNotification(types.NOTIFY_ERROR, err.Error())
	}
}

func (sr *sdlRender) UpdateNotes(tile types.Tile) {
	if !notes.mutex.TryLock() {
		// only run once
		return
	}
	if notes.ipc == nil {
		notes.mutex.Unlock()
		return
	}
	notes.mutex.Unlock()

	err := notes.ipc.Send(&dispatcher.IpcMessageT{
		EventName: "notesUpdatePaths",
		Parameters: map[string]string{
			"projectRoot": findProjectRoot(tile.Pwd()),
			"userNotes":   userDocs(tile, "notes"),
		},
	})
	if err != nil {
		sr.DisplayNotification(types.NOTIFY_ERROR, err.Error())
	}
}

func findProjectRoot(cwd string) string {
	pwd := cwd
	home, _ := os.UserHomeDir()
	for {
		if _, err := os.Stat(filepath.Join(cwd, ".git")); err == nil {
			return pwd
		}
		parent := filepath.Dir(cwd)
		if parent == pwd || parent == home {
			return ""
		}
		pwd = parent
	}
}
