package menuhyperlink

import (
	"os"

	"github.com/lmorg/ttyphoon/types"
)

func menuItemsSchemaFile(link LinkT, menuItems []types.MenuItem) []types.MenuItem {
	if link.Scheme() != "file" {
		return menuItems
	}

	f, err := os.Open(schemaOrPath(link))
	if err != nil {
		link.Renderer().DisplayNotification(types.NOTIFY_ERROR, err.Error())
		return menuItems
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		link.Renderer().DisplayNotification(types.NOTIFY_ERROR, err.Error())
		return menuItems
	}

	if info.IsDir() || info.Size() > _CONTENTS_CLIP_MAX {
		return menuItems
	}

	menuItems = append(menuItems,
		types.MenuItem{
			Title: "Copy contents to clipboard",
			Fn:    func() { copyContentsToClipboard(link.Renderer(), schemaOrPath(link)) },
			Icon:  0xf0c6,
		},
	)

	/*if strings.HasSuffix(el.url, ".md") {
		menuItems = append(menuItems, types.MenuItem{
			Title: "Open in markdown viewer",
			Fn:    func() { openMarkdownViewer(el) },
			Icon:  0xf1ea,
		})
	}*/

	return menuItems
}

/*func openMarkdownViewer(el *ElementHyperlink) {
	windowStyle := dispatcher.NewWindowStyle()
	//windowStyle.Pos = types.XY{}
	//x, y := el.renderer.GetWindowMeta().(*sdl.Window).GetSize()
	//windowStyle.Size = types.XY{X: x, Y: y}
	//windowStyle.Title = string(el.label)

	parameters := &dispatcher.PMarkdownT{Path: el.link}

	_, _ = dispatcher.DisplayWindow(dispatcher.WindowMarkdown, windowStyle, parameters, func(msg *dispatcher.IpcMessageT) {
		if msg.Error != nil {
			el.renderer.DisplayNotification(types.NOTIFY_ERROR, msg.Error.Error())
		}
	})
}*/
