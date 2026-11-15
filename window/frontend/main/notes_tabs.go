package mainwindow

import "github.com/lmorg/ttyphoon/utils/notes"

// NotesOpenTabT is an open document tab as the frontend names it: a path,
// possibly using a Notes alias, and the project root that $PROJECT meant.
type NotesOpenTabT struct {
	Path    string `json:"path"`
	Project string `json:"project"`
}

// NotesOpenViewT is a view's tabs as the frontend saves them.
type NotesOpenViewT struct {
	ID     string          `json:"id"`
	Tabs   []NotesOpenTabT `json:"tabs"`
	Active int             `json:"active"`
}

// NotesOpenLayoutT is a workspace's views as the frontend saves them. Layout
// is the frontend's layout tree as JSON, naming views by ID.
type NotesOpenLayoutT struct {
	Views   []NotesOpenViewT `json:"views"`
	Layout  string           `json:"layout"`
	Focused string           `json:"focused"`
}

// NotesSavedViewT is a remembered view's tabs, as paths in the current context.
type NotesSavedViewT struct {
	ID     string   `json:"id"`
	Files  []string `json:"files"`
	Active int      `json:"active"`
}

type NotesOpenTabsT struct {
	Workspace string            `json:"workspace"`
	Views     []NotesSavedViewT `json:"views"`
	Layout    string            `json:"layout"`
	Focused   string            `json:"focused"`
}

// GetNotesOpenTabs returns the current workspace's remembered views and tabs,
// with paths in the current context using Notes aliases, as history does.
func (a *WApp) GetNotesOpenTabs() NotesOpenTabsT {
	workspace := a.currentNotesContext().GroupName
	tabs := notes.GetOpenTabs(workspace)
	views := make([]NotesSavedViewT, 0, len(tabs.Views))
	for _, view := range tabs.Views {
		files := make([]string, 0, len(view.Files))
		active := 0
		for i, filename := range view.Files {
			path := a.notesPathForHistory(filename)
			if path == "" {
				continue
			}
			if i == view.Active {
				active = len(files)
			}
			files = append(files, path)
		}
		if len(files) > 0 {
			views = append(views, NotesSavedViewT{ID: view.ID, Files: files, Active: active})
		}
	}
	return NotesOpenTabsT{Workspace: workspace, Views: views, Layout: tabs.Layout, Focused: tabs.Focused}
}

// SetNotesOpenTabs remembers a workspace's views and tabs. It is ignored
// unless the workspace is the current one, so a save sent before a workspace
// change cannot overwrite the tabs of the workspace that replaced it.
func (a *WApp) SetNotesOpenTabs(workspace string, layout NotesOpenLayoutT) bool {
	if workspace != a.currentNotesContext().GroupName {
		return false
	}
	views := make([]notes.OpenViewT, 0, len(layout.Views))
	for _, view := range layout.Views {
		files := make([]string, 0, len(view.Tabs))
		active := 0
		for i, tab := range view.Tabs {
			if tab.Path == "" {
				continue
			}
			if i == view.Active {
				active = len(files)
			}
			files = append(files, a.filePathWithProject(tab.Path, tab.Project))
		}
		views = append(views, notes.OpenViewT{ID: view.ID, Files: files, Active: active})
	}
	notes.SetOpenTabs(workspace, notes.OpenTabsT{Views: views, Layout: layout.Layout, Focused: layout.Focused})
	return true
}
