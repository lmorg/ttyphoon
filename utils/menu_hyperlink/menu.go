package menuhyperlink

import (
	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/types"
)

func MenuItems(link LinkT) []types.MenuItem {
	menuItems := []types.MenuItem{
		{
			Title: "Copy link to clipboard",
			Fn:    func() { copyLinkToClipboard(link.Renderer(), schemaOrPath(link)) },
			Icon:  0xf0c1,
		},
		{
			Title: "Copy text to clipboard",
			Fn:    func() { copyLinkToClipboard(link.Renderer(), link.Label()) },
			Icon:  0xf0c5,
		},
	}

	menuItems = menuItemsSchemaHttp(link, menuItems)
	menuItems = menuItemsSchemaFile(link, menuItems)

	apps, cmds := config.Config.Terminal.Widgets.AutoHyperlink.OpenAgents.MenuItems(link.Scheme())
	for i := range apps {
		menuItems = append(menuItems,
			types.MenuItem{
				Title: "Open link with " + apps[i],
				Fn:    func() { OpenWith(link, cmds[i]) },
				Icon:  0xf08e,
			},
		)
	}
	menuItems = append(menuItems, []types.MenuItem{
		{
			Title: "Write link to shell",
			Fn:    func() { link.Renderer().ActiveTile().GetTerm().Reply([]byte(schemaOrPath(link))) },
			Icon:  0xf120,
		},
	}...,
	)

	return menuItems
}
