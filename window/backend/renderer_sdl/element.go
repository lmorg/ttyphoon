package rendersdl

import (
	"github.com/lmorg/mxtty/types"
	"github.com/lmorg/mxtty/window/backend/renderer_sdl/element_codeblock"
	"github.com/lmorg/mxtty/window/backend/renderer_sdl/element_hyperlink"
	"github.com/lmorg/mxtty/window/backend/renderer_sdl/element_image"
	"github.com/lmorg/mxtty/window/backend/renderer_sdl/element_sixel"
	"github.com/lmorg/mxtty/window/backend/renderer_sdl/element_table"
)

func (sr *sdlRender) NewElement(tile types.Tile, id types.ElementID) types.Element {
	switch id {
	case types.ELEMENT_ID_IMAGE:
		return element_image.New(sr, tile, sr.loadImage)

	case types.ELEMENT_ID_SIXEL:
		return element_sixel.New(sr, tile, sr.loadImage)

	case types.ELEMENT_ID_CSV:
		return element_table.NewCsv(sr, tile)

	case types.ELEMENT_ID_MARKDOWN_TABLE:
		return element_table.NewMarkdown(sr, tile)

	case types.ELEMENT_ID_HYPERLINK:
		return element_hyperlink.New(sr, tile)

	case types.ELEMENT_ID_CODEBLOCK:
		return element_codeblock.New(sr, tile)

	default:
		return nil
	}
}
