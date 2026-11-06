package mainwindow

import (
	"github.com/lmorg/ttyphoon/utils/notes"
)

func (a *WApp) GetProjectCache() *notes.ProjectCacheT {
	return notes.GetProjectCache(a.GetCurrentProject())
}

func (a *WApp) GetNotesLayoutSizing() notes.LayoutSizing {
	return notes.GetLayoutSizing()
}

func (a *WApp) SetNotesLayoutRatio(name string, ratio float64) error {
	return notes.SetLayoutRatio(name, ratio)
}

func (a *WApp) SetProjectCache(ptr *notes.ProjectCacheT) {
	notes.SetProjectCache(a.GetCurrentProject(), ptr)
}

func (a *WApp) GetDocumentCache(filename string) *notes.DocumentCacheT {
	return notes.GetDocumentCache(a.GetCurrentProject(), filename)
}

func (a *WApp) SetDocumentCache(filename string, ptr *notes.DocumentCacheT) {
	notes.SetDocumentCache(a.GetCurrentProject(), filename, ptr)
}

func (a *WApp) GetNotesColumnWidths(filename, view string, headings []string, wrapped bool) []float64 {
	return notes.GetColumnWidths(filename, view, headings, wrapped)
}

func (a *WApp) SetNotesColumnWidths(filename, view string, headings []string, wrapped bool, widths []float64) {
	notes.SetColumnWidths(filename, view, headings, wrapped, widths)
}
