package mainwindow

import "github.com/lmorg/ttyphoon/utils/notes"

// NotesOpenTabT is an open document tab as the frontend names it: a path,
// possibly using a Notes alias, and the project root that $PROJECT meant.
type NotesOpenTabT struct {
	Path    string `json:"path"`
	Project string `json:"project"`
}

type NotesOpenTabsT struct {
	Workspace string   `json:"workspace"`
	Files     []string `json:"files"`
	Active    int      `json:"active"`
}

// GetNotesOpenTabs returns the current workspace's remembered tabs, with
// paths in the current context using Notes aliases, as history does.
func (a *WApp) GetNotesOpenTabs() NotesOpenTabsT {
	workspace := a.currentNotesContext().GroupName
	tabs := notes.GetOpenTabs(workspace)
	files := make([]string, 0, len(tabs.Files))
	for _, filename := range tabs.Files {
		if path := a.notesPathForHistory(filename); path != "" {
			files = append(files, path)
		}
	}
	return NotesOpenTabsT{Workspace: workspace, Files: files, Active: min(tabs.Active, max(len(files)-1, 0))}
}

// SetNotesOpenTabs remembers a workspace's tabs. It is ignored unless the
// workspace is the current one, so a save sent before a workspace change
// cannot overwrite the tabs of the workspace that replaced it.
func (a *WApp) SetNotesOpenTabs(workspace string, tabs []NotesOpenTabT, active int) bool {
	if workspace != a.currentNotesContext().GroupName {
		return false
	}
	files := make([]string, 0, len(tabs))
	for _, tab := range tabs {
		if tab.Path != "" {
			files = append(files, a.filePathWithProject(tab.Path, tab.Project))
		}
	}
	notes.SetOpenTabs(workspace, notes.OpenTabsT{Files: files, Active: active})
	return true
}
