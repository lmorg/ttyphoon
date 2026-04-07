package menuhyperlink

import (
	"strings"

	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/types"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type link struct {
	renderer types.Renderer
	url      string
	scheme   string
	path     string
	label    string

	// fileName is just the trailing file name after the slash
	fileName string
	// filePath is just the path leading up to the file name. It should be an absolute path and it should include the trailing slash. If file in in the root then it should be `/`
	filePath string
}

func makeLink(renderer types.Renderer, url, label string) *link {
	url = strings.TrimSpace(url)
	label = strings.TrimSpace(label)
	if label == "" {
		label = url
	}

	scheme := ""
	path := ""
	split := strings.SplitN(url, "://", 2)
	if len(split) == 2 {
		scheme = strings.ToLower(strings.TrimSpace(split[0]))
		path = split[1]
	}

	link := &link{
		renderer: renderer,
		url:      url,
		scheme:   scheme,
		path:     path,
		label:    label,
	}

	// Split fileName and filePath for file scheme URLs
	if link.scheme == "file" {
		lastSlash := strings.LastIndex(link.path, "/")
		if lastSlash >= 0 {
			link.filePath = link.path[:lastSlash+1]
			link.fileName = link.path[lastSlash+1:]
		} else {
			link.filePath = "/"
			link.fileName = link.path
		}
	}

	return link
}

func MenuItems(renderer types.Renderer, url, label string) []types.MenuItem {
	link := makeLink(renderer, url, label)
	if link.url == "" {
		return []types.MenuItem{}
	}

	openTarget := "link"
	writeTargetTitle := "Write link to shell"
	menuItems := []types.MenuItem{}

	if link.scheme == "file" {
		openTarget = "file"
		writeTargetTitle = "Write file path to shell"

		menuItems = []types.MenuItem{
			{
				Title: "Copy file and path to clipboard",
				Fn:    func() { copyLinkToClipboard(link.renderer, link.filePath+link.fileName) },
				Icon:  0xf0c1,
			},
			{
				Title: "Copy file name to clipboard",
				Fn:    func() { copyLinkToClipboard(link.renderer, link.fileName) },
				Icon:  0xf0c5,
			},
		}
	} else {
		menuItems = []types.MenuItem{
			{
				Title: "Copy link to clipboard",
				Fn:    func() { copyLinkToClipboard(link.renderer, schemaOrPath(link)) },
				Icon:  0xf0c1,
			},
			{
				Title: "Copy text to clipboard",
				Fn:    func() { copyLinkToClipboard(link.renderer, link.label) },
				Icon:  0xf0c5,
			},
		}
	}

	menuItems = menuItemsSchemaHttp(link, menuItems)
	menuItems = menuItemsSchemaFile(link, menuItems)

	apps, cmds := config.Config.Terminal.Widgets.AutoHyperlink.OpenAgents.MenuItems(link.scheme)
	for i := range apps {
		menuItems = append(menuItems,
			types.MenuItem{
				Title: "Open " + openTarget + " with " + apps[i],
				Fn:    func() { OpenWith(link.renderer, link.url, link.label, cmds[i]) },
				Icon:  0xf08e,
			},
		)
	}
	menuItems = append(menuItems, []types.MenuItem{
		{
			Title: writeTargetTitle,
			Fn:    func() { link.renderer.ActiveTile().GetTerm().Reply([]byte(schemaOrPath(link))) },
			Icon:  0xf120,
		},
	}...,
	)

	// Add file-specific actions for file scheme
	if link.scheme == "file" {
		filePath := link.filePath + link.fileName
		fileName := link.fileName

		menuItems = append(menuItems, []types.MenuItem{
			{
				Title: types.MENU_SEPARATOR,
			},
			{
				Title: "Rename file",
				Fn: func() {
					runtime.EventsEmit(link.renderer.GetContext(), "fileActionDialog", map[string]string{
						"action":   "rename",
						"filePath": filePath,
						"fileName": fileName,
					})
				},
				Icon: 0xf044,
			},
			{
				Title: "Delete file",
				Fn: func() {
					runtime.EventsEmit(link.renderer.GetContext(), "fileActionDialog", map[string]string{
						"action":   "delete",
						"filePath": filePath,
						"fileName": fileName,
					})
				},
				Icon: 0xf1f8,
			},
		}...,
		)
	}

	return menuItems
}
