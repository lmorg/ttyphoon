package notes

import (
	"github.com/lmorg/ttyphoon/utils/cache"
)

// MaxOpenTabs caps how many open document tabs are remembered per view.
const MaxOpenTabs = 50

// MaxOpenViews caps how many split views are remembered per workspace.
const MaxOpenViews = 16

// OpenViewT records one view's tabs: the expanded filenames in tab order and
// the index of the tab it shows.
type OpenViewT struct {
	ID     string   `json:"id"`
	Files  []string `json:"files"`
	Active int      `json:"active"`
}

// OpenTabsT records a workspace's open document tabs (ADR 0100 question 8)
// and how its views are split (Stage 8). Layout is the frontend's layout
// tree, kept as opaque JSON whose leaves name views by ID. Files and Active
// are the single view saved before views were split, and are read as one
// view.
type OpenTabsT struct {
	Files   []string    `json:"files,omitempty"`
	Active  int         `json:"active,omitempty"`
	Views   []OpenViewT `json:"views,omitempty"`
	Layout  string      `json:"layout,omitempty"`
	Focused string      `json:"focused,omitempty"`
}

func openTabsKey(workspace string) string {
	return "tabs:workspace:" + workspace
}

func GetOpenTabs(workspace string) OpenTabsT {
	var tabs OpenTabsT
	cache.Read(cache.NS_NOTESW_HISTORY, openTabsKey(workspace), &tabs)
	if len(tabs.Views) == 0 && len(tabs.Files) > 0 {
		tabs.Views = []OpenViewT{{Files: tabs.Files, Active: tabs.Active}}
	}
	tabs.Files, tabs.Active = nil, 0
	return normaliseOpenTabs(tabs)
}

func SetOpenTabs(workspace string, tabs OpenTabsT) {
	tabs.Files, tabs.Active = nil, 0
	tabs = normaliseOpenTabs(tabs)
	cache.Write(cache.NS_NOTESW_HISTORY, openTabsKey(workspace), &tabs, cache.Days(30))
}

func normaliseOpenTabs(tabs OpenTabsT) OpenTabsT {
	views := make([]OpenViewT, 0, len(tabs.Views))
	for _, view := range tabs.Views {
		if len(view.Files) == 0 {
			continue
		}
		if len(view.Files) > MaxOpenTabs {
			view.Files = view.Files[:MaxOpenTabs]
		}
		if view.Active < 0 || view.Active >= len(view.Files) {
			view.Active = 0
		}
		views = append(views, view)
		if len(views) == MaxOpenViews {
			break
		}
	}
	tabs.Views = views
	return tabs
}
