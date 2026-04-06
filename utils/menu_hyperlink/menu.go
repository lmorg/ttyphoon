package menuhyperlink

import (
	"strings"

	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/types"
)

type linkT interface {
	Renderer() types.Renderer
	Url() string
	Scheme() string
	Path() string
	Label() string
}

type link struct {
	renderer types.Renderer
	url      string
	scheme   string
	path     string
	label    string
}

func (l *link) Renderer() types.Renderer { return l.renderer }
func (l *link) Url() string              { return l.url }
func (l *link) Scheme() string           { return l.scheme }
func (l *link) Path() string             { return l.path }
func (l *link) Label() string            { return l.label }

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

	return &link{
		renderer: renderer,
		url:      url,
		scheme:   scheme,
		path:     path,
		label:    label,
	}
}

func MenuItems(renderer types.Renderer, url, label string) []types.MenuItem {
	link := makeLink(renderer, url, label)
	if link.url == "" {
		return []types.MenuItem{}
	}

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
				Fn:    func() { OpenWith(link.Renderer(), link.Url(), link.Label(), cmds[i]) },
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
