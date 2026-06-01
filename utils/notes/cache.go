package notes

import (
	"fmt"

	"github.com/lmorg/ttyphoon/utils/cache"
)

// Project-wide cache

type ProjectCacheT struct {
	FileListCollapsed map[string][]string // directories that have been collapsed. The map key will be the project root. And slice will be a list of directories collapsed
	LastDocument      string              // filename of last document open in that window/tab. "" means don't reopen any docs
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
