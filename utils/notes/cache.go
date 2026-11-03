package notes

import (
	"fmt"
	"math"
	"sync"

	"github.com/lmorg/ttyphoon/utils/cache"
)

type LayoutSizing struct {
	FileListRatio float64 `json:"fileListRatio"`
	NotesRatio    float64 `json:"notesRatio"`
	ToolsRatio    float64 `json:"toolsRatio"`
}

var layoutSizingMu sync.Mutex

func GetLayoutSizing() LayoutSizing {
	return getLayoutSizing("application")
}

func getLayoutSizing(key string) LayoutSizing {
	layoutSizingMu.Lock()
	defer layoutSizingMu.Unlock()
	var sizing LayoutSizing
	cache.Read(cache.NS_NOTESW_LAYOUT, key, &sizing)
	return sizing
}

func SetLayoutRatio(name string, ratio float64) error {
	return setLayoutRatio("application", name, ratio)
}

func setLayoutRatio(key, name string, ratio float64) error {
	if math.IsNaN(ratio) || math.IsInf(ratio, 0) || ratio < 0 || ratio >= 1 {
		return fmt.Errorf("invalid layout ratio %v", ratio)
	}
	layoutSizingMu.Lock()
	defer layoutSizingMu.Unlock()
	var sizing LayoutSizing
	cache.Read(cache.NS_NOTESW_LAYOUT, key, &sizing)
	switch name {
	case "fileListRatio":
		sizing.FileListRatio = ratio
	case "notesRatio":
		sizing.NotesRatio = ratio
	case "toolsRatio":
		sizing.ToolsRatio = ratio
	default:
		return fmt.Errorf("unknown layout size %q", name)
	}
	cache.Write(cache.NS_NOTESW_LAYOUT, key, &sizing, cache.Days(365))
	return nil
}

// Project-wide cache

type ProjectCacheT struct {
	FileListCollapsed map[string][]string // directories that have been collapsed. The map key will be the project root. And slice will be a list of directories collapsed
}

func GetProjectCache(projectRoot string) *ProjectCacheT {
	ptr := &ProjectCacheT{
		FileListCollapsed: map[string][]string{},
	}
	cache.Read(cache.NS_NOTESW_PROJECT, projectRoot, ptr)
	return ptr
}

func SetProjectCache(projectRoot string, ptr *ProjectCacheT) {
	if ptr == nil {
		return
	}

	cache.Write(cache.NS_NOTESW_PROJECT, projectRoot, ptr, cache.Days(365))
}

// Document cache

type DocumentCacheT struct {
	DocumentTab string // which mode (eg View, Run, Hex) is selected. "" means the first tab
	ToolsOpen   bool   // default false means Tools panel is minimized
	ToolsTab    string // which tab is selected in Tools. "" means the first tab
	WordWrap    bool   // whether editor word wrap was enabled for this document
}

func GetDocumentCache(projectRoot, filename string) *DocumentCacheT {
	ptr := new(DocumentCacheT)
	cache.Read(cache.NS_NOTESW_DOCUMENT, fmt.Sprintf("%s::%s", projectRoot, filename), ptr)
	return ptr
}

func SetDocumentCache(projectRoot, filename string, ptr *DocumentCacheT) {
	if ptr == nil {
		return
	}

	cache.Write(cache.NS_NOTESW_DOCUMENT, fmt.Sprintf("%s::%s", projectRoot, filename), ptr, cache.Days(30))
}
