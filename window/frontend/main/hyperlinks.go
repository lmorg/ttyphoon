package mainwindow

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/utils/hyperlink_tooltip"
	menuhyperlink "github.com/lmorg/ttyphoon/utils/menu_hyperlink"
	renderwebkit "github.com/lmorg/ttyphoon/window/backend/renderer_webkit"
)

func (a *WApp) GetHyperlinkTooltip(rawURL string) (string, error) {
	scheme, target, found := strings.Cut(rawURL, ":")
	if !found || scheme == "" {
		return "", fmt.Errorf("hyperlink tooltip URL is missing a scheme")
	}
	parsed, err := url.Parse(scheme + ":")
	if err != nil {
		return "", fmt.Errorf("invalid hyperlink tooltip scheme: %w", err)
	}
	if parsed.Scheme == "" || strings.IndexFunc(rawURL, func(r rune) bool { return r < ' ' || r == 127 }) >= 0 {
		return "", fmt.Errorf("invalid hyperlink tooltip URL")
	}
	return hyperlink_tooltip.HTML(strings.ToLower(parsed.Scheme), strings.TrimPrefix(target, "//")), nil
}

func (a *WApp) hyperlinkMenuItems(url, text string) []types.MenuItem {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return nil
	}

	return menuhyperlink.MenuItems(renderer, url, text)
}

func (a *WApp) GetHyperlinkMenuActions(url, text string) []map[string]any {
	menuItems := a.hyperlinkMenuItems(url, text)
	out := make([]map[string]any, 0, len(menuItems))

	for i := range menuItems {
		out = append(out, map[string]any{
			"title":  menuItems[i].Title,
			"icon":   menuItems[i].Icon,
			"action": strconv.Itoa(i),
		})
	}

	return out
}

func (a *WApp) RunHyperlinkMenuAction(url, text, action string) {
	menuItems := a.hyperlinkMenuItems(url, text)
	if len(menuItems) == 0 {
		return
	}

	index, err := strconv.Atoi(strings.TrimSpace(action))
	if err != nil || index < 0 || index >= len(menuItems) {
		return
	}

	// Execute the menu item callback if it exists
	if menuItems[index].Fn != nil {
		menuItems[index].Fn()
	}
}

func (a *WApp) DisplayHyperlinkMenu(url, text string) {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return
	}

	menu := renderer.NewContextMenu()
	menu.Append(a.hyperlinkMenuItems(url, text)...)
	menu.DisplayMenu("Hyperlink action", true)
}

func (a *WApp) HyperlinkOpenWithDefault(url string) {
	renderer, ok := renderwebkit.CurrentRenderer()
	if !ok {
		return
	}

	menuhyperlink.OpenWithDefault(renderer, url, "")
}

func (a *WApp) OpenURLInNewWindow(url string) error {
	if a.uiRuntime == nil {
		return fmt.Errorf("desktop runtime is unavailable")
	}
	return a.uiRuntime.OpenURLInNewWindow(url)
}

func (a *WApp) GetCustomRegexp() []map[string]string {
	var result []map[string]string
	for _, custom := range config.Config.Terminal.Widgets.AutoHyperlink.CustomRegexp {
		if custom.Rx == nil {
			continue
		}
		result = append(result, map[string]string{
			"pattern": custom.Rx.String(),
			"link":    custom.Link,
		})
	}
	return result
}
