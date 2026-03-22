package rendererwebkit

import (
	"github.com/lmorg/ttyphoon/types"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type menuCallbacks struct {
	highlight types.MenuCallbackT
	selectFn  types.MenuCallbackT
	cancel    types.MenuCallbackT
}

type terminalMenuEvent struct {
	MenuID  int      `json:"menuId"`
	Title   string   `json:"title"`
	Options []string `json:"options"`
	Icons   []rune   `json:"icons"`
}

func (wr *webkitRender) DisplayMenu(title string, items []string, highlight types.MenuCallbackT, ok types.MenuCallbackT, cancel types.MenuCallbackT) {
	wr.openMenu(title, items, nil, highlight, ok, cancel)
}

func (wr *webkitRender) openMenu(title string, items []string, icons []rune, highlight types.MenuCallbackT, ok types.MenuCallbackT, cancel types.MenuCallbackT) {
	if len(items) == 0 {
		return
	}

	if highlight == nil {
		highlight = func(int) {}
	}
	if ok == nil {
		ok = func(int) {}
	}
	if cancel == nil {
		cancel = func(int) {}
	}

	wr.menuMu.Lock()
	wr.menuNextID++
	menuID := wr.menuNextID
	wr.menuCallbacks[menuID] = menuCallbacks{highlight: highlight, selectFn: ok, cancel: cancel}
	wr.menuMu.Unlock()

	if wr.wapp == nil {
		wr.MenuCancel(menuID, -1)
		return
	}

	runtime.EventsEmit(wr.wapp, "terminalListMenu", terminalMenuEvent{
		MenuID:  menuID,
		Title:   title,
		Options: append([]string(nil), items...),
		Icons:   append([]rune(nil), icons...),
	})
}

func (wr *webkitRender) NewContextMenu() types.ContextMenu {
	menu := &contextMenuStub{renderer: wr}
	return menu
}

func (wr *webkitRender) AddToContextMenu(menuItems ...types.MenuItem) {
	if wr.contextMenu == nil {
		wr.contextMenu = wr.NewContextMenu()
	}

	for i := range menuItems {
		if menuItems[i].Highlight != nil {
			menuItems[i].WebkitContextHighlightPersistent = true
		}
	}

	wr.contextMenu.Append(menuItems...)
}

func (wr *webkitRender) MenuHighlight(menuID int, index int) {
	wr.menuMu.Lock()
	callbacks, ok := wr.menuCallbacks[menuID]
	wr.menuMu.Unlock()

	if !ok || callbacks.highlight == nil {
		return
	}

	callbacks.highlight(index)
}

func (wr *webkitRender) MenuSelect(menuID int, index int) {
	wr.menuMu.Lock()
	callbacks, ok := wr.menuCallbacks[menuID]
	if ok {
		delete(wr.menuCallbacks, menuID)
	}
	wr.menuMu.Unlock()

	if !ok || callbacks.selectFn == nil {
		return
	}

	callbacks.selectFn(index)
}

func (wr *webkitRender) MenuCancel(menuID int, index int) {
	wr.menuMu.Lock()
	callbacks, ok := wr.menuCallbacks[menuID]
	if ok {
		delete(wr.menuCallbacks, menuID)
	}
	wr.menuMu.Unlock()

	if !ok || callbacks.cancel == nil {
		return
	}

	callbacks.cancel(index)
}

type contextMenuStub struct {
	items    []types.MenuItem
	renderer *webkitRender
}

func (cms *contextMenuStub) Append(items ...types.MenuItem) {
	cms.items = append(cms.items, items...)
}

func (cms *contextMenuStub) DisplayMenu(title string) {
	if cms.renderer == nil {
		return
	}

	cms.renderer.openMenu(title, cms.Options(), cms.Icons(),
		func(i int) { cms.Highlight(i) },
		func(i int) { cms.Callback(i) },
		func(i int) { cms.Cancel(i) },
	)
}

func (cms *contextMenuStub) Options() []string {
	options := make([]string, len(cms.items))
	for i := range cms.items {
		options[i] = cms.items[i].Title
	}
	return options
}

func (cms *contextMenuStub) Icons() []rune {
	icons := make([]rune, len(cms.items))
	for i := range cms.items {
		icons[i] = cms.items[i].Icon
	}
	return icons
}

func (cms *contextMenuStub) Highlight(i int) {
	cms.clearHover()

	if i < 0 || i >= len(cms.items) {
		return
	}

	if cms.items[i].Highlight == nil {
		return
	}

	hoverFn := cms.items[i].Highlight()
	if cms.renderer == nil {
		return
	}

	if cms.items[i].WebkitContextHighlightPersistent {
		cms.renderer.setMenuHover(hoverFn, nil, true)
		cms.renderer.TriggerRedraw()
		return
	}

	cms.renderer.setMenuHover(nil, hoverFn, false)
	cms.renderer.TriggerRedraw()
}

func (cms *contextMenuStub) Callback(i int) {
	cms.clearHover()

	if i < 0 || i >= len(cms.items) {
		return
	}
	if cms.items[i].Fn != nil {
		cms.items[i].Fn()
	}
}

func (cms *contextMenuStub) Cancel(_ int) {
	cms.clearHover()
}

func (cms *contextMenuStub) MenuItems() []types.MenuItem {
	return append([]types.MenuItem(nil), cms.items...)
}

func (cms *contextMenuStub) clearHover() {
	if cms.renderer != nil {
		cms.renderer.clearMenuHover()
	}
}

func (wr *webkitRender) setMenuHover(drawFn, clearFn func(), drawn bool) {
	wr.menuMu.Lock()
	wr.menuHoverFn = drawFn
	wr.menuHoverClear = clearFn
	wr.menuHoverDrawn = drawn
	wr.menuMu.Unlock()
}

func (wr *webkitRender) clearMenuHover() {
	wr.menuMu.Lock()
	drawFn := wr.menuHoverFn
	clearFn := wr.menuHoverClear
	drawn := wr.menuHoverDrawn
	wr.menuHoverFn = nil
	wr.menuHoverClear = nil
	wr.menuHoverDrawn = false
	wr.menuMu.Unlock()

	if !drawn && clearFn != nil {
		clearFn()
	}

	if drawn && drawFn != nil {
		wr.TriggerRedraw()
	}
}

func (wr *webkitRender) applyMenuHover() {
	wr.menuMu.Lock()
	drawFn := wr.menuHoverFn
	drawn := wr.menuHoverDrawn
	wr.menuMu.Unlock()

	if drawn && drawFn != nil {
		drawFn()
	}
}
