package menuhyperlink

import (
	"fmt"

	"github.com/lmorg/ttyphoon/ai"
	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/types"
)

func menuItemsSchemaHttp(link *link, menuItems []types.MenuItem) []types.MenuItem {
	if link.scheme != "http" {
		return menuItems
	}

	agt := agent.Get(link.renderer.ActiveTile().Id())
	agt.Meta = &agent.Meta{}
	menuItems = append(menuItems, types.MenuItem{
		Title: fmt.Sprintf("Summarize hyperlink (%s)", agt.ServiceName()),
		Fn: func() {
			ai.AskAI(agt, fmt.Sprintf("Can you summarize the contents of this web page: %s\n Do NOT to check other websites nor use any search engines.", link.url))
		},
		Icon: 0xf544,
	})

	return menuItems
}
