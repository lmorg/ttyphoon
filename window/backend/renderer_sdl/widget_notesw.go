//go:build ignore
// +build ignore

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

func (sr *sdlRender) startNotes(tile types.Tile, filename, content string) bool {
	if !notes.mutex.TryLock() {
		// only run once
		return true
	}
	defer notes.mutex.Unlock()
	if notes.ipc != nil {
		return false
	}

	windowStyle := dispatcher.NewWindowStyle()
	/*windowStyle.Pos = types.XY{}
	x, y := sr.window.GetSize()
	windowStyle.Size = types.XY{X: x, Y: y}
	windowStyle.Title = tile.GroupName()
	windowStyle.AlwaysOnTop = config.Config.NotesWindow.AlwaysOnTop*/

	parameters := &dispatcher.PNotesT{
		ProjectRoot: findProjectRoot(tile.Pwd()),
		UserNotes:   userDocs(tile, "notes"),
		Title:       tile.GroupName(),
		Filename:    filename,
		Content:     content,
	}

	notes.ipc, _ = dispatcher.DisplayWindow(dispatcher.WindowNotes, windowStyle, parameters, func(msg *dispatcher.IpcMessageT) {
		if msg.Error != nil {
			sr.DisplayNotification(types.NOTIFY_ERROR, msg.Error.Error())
		} else {
			switch msg.EventName {
			case "focus":
				sr.TriggerDeallocation(sr.window.Raise)
			case "noteRunTerminal":
				sr.TriggerDeallocation(sr.window.Raise)
				b := []byte(msg.Parameters["code"])
				err := sr.tmux.ActivePane().Write(b)
				if err != nil {
					sr.DisplayNotification(types.NOTIFY_ERROR, err.Error())
				}
			}
		}
	})
	return true
}

func (sr *sdlRender) openNotes() {
	tile := sr.termWin.Active
	if sr.startNotes(tile, "", "") {
		return
	}

	err := notes.ipc.Send(&dispatcher.IpcMessageT{
		EventName: "notesFocus",
		Parameters: map[string]string{
			"projectRoot": findProjectRoot(tile.Pwd()),
			"userNotes":   userDocs(tile, "notes"),
			"title":       tile.GroupName(),
		},
	})
	if err != nil {
		sr.DisplayNotification(types.NOTIFY_ERROR, err.Error())
	}
}

func (sr *sdlRender) toggleNotes() {
	if notes.ipc == nil {
		sr.startNotes(sr.termWin.Active, "", "")
		return
	}

	err := notes.ipc.Send(&dispatcher.IpcMessageT{
		EventName: "notesToggleShowHide",
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
			"title":       tile.GroupName(),
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

func (sr *sdlRender) NotesCreateAndOpen(filename, content string) {
	if sr.startNotes(sr.termWin.Active, filename, content) {
		return
	}

	err := notes.ipc.Send(&dispatcher.IpcMessageT{
		EventName: "notesCreateAndOpen",
		Parameters: map[string]string{
			"filename": filename,
			"contents": content,
		},
	})
	if err != nil {
		sr.DisplayNotification(types.NOTIFY_ERROR, err.Error())
	}
}
