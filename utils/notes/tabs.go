package notes

import (
	"github.com/lmorg/ttyphoon/utils/cache"
)

// MaxOpenTabs caps how many open document tabs are remembered per workspace.
const MaxOpenTabs = 50

// OpenTabsT records a workspace's open document tabs (ADR 0100 question 8):
// the expanded filenames in tab order and the index of the focused tab.
type OpenTabsT struct {
	Files  []string `json:"files"`
	Active int      `json:"active"`
}

func openTabsKey(workspace string) string {
	return "tabs:workspace:" + workspace
}

func GetOpenTabs(workspace string) OpenTabsT {
	var tabs OpenTabsT
	cache.Read(cache.NS_NOTESW_HISTORY, openTabsKey(workspace), &tabs)
	if tabs.Active < 0 || tabs.Active >= len(tabs.Files) {
		tabs.Active = 0
	}
	return tabs
}

func SetOpenTabs(workspace string, tabs OpenTabsT) {
	if len(tabs.Files) > MaxOpenTabs {
		tabs.Files = tabs.Files[:MaxOpenTabs]
	}
	if tabs.Active < 0 || tabs.Active >= len(tabs.Files) {
		tabs.Active = 0
	}
	cache.Write(cache.NS_NOTESW_HISTORY, openTabsKey(workspace), &tabs, cache.Days(30))
}
