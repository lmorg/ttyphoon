package menuhyperlink

import (
	"fmt"

	"github.com/lmorg/ttyphoon/ai"
	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/types"
)

func menuItemsSchemaHttp(link linkT, menuItems []types.MenuItem) []types.MenuItem {
	if link.Scheme() != "http" {
		return menuItems
	}

	agt := agent.Get(link.Renderer().ActiveTile().Id())
	agt.Meta = &agent.Meta{}
	menuItems = append(menuItems, types.MenuItem{
		Title: fmt.Sprintf("Summarize hyperlink (%s)", agt.ServiceName()),
		Fn: func() {
			ai.AskAI(agt, fmt.Sprintf("Can you summarize the contents of this web page: %s\n Do NOT to check other websites nor use any search engines.", link.Url()))
		},
		Icon: 0xf544,
	})

	return menuItems
}
