package rendererwebkit

import (
	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/window/backend/renderer_sdl/element_codeblock"
	"github.com/lmorg/ttyphoon/window/backend/renderer_sdl/element_hyperlink"
	"github.com/lmorg/ttyphoon/window/backend/renderer_sdl/element_table"
)

func (wr *webkitRender) NewElement(tile types.Tile, id types.ElementID) types.Element {
	switch id {
	//case types.ELEMENT_ID_IMAGE:
	//	return element_image.NewBitmap(sr, tile, sr.loadImage)

	//case types.ELEMENT_ID_SIXEL:
	//	return element_image.NewSixel(sr, tile, sr.loadImage)

	case types.ELEMENT_ID_CSV:
		return element_table.NewCsv(wr, tile)

	case types.ELEMENT_ID_MARKDOWN_TABLE:
		return element_table.NewMarkdown(wr, tile)

	case types.ELEMENT_ID_HYPERLINK:
		return element_hyperlink.New(wr, tile)

	case types.ELEMENT_ID_CODEBLOCK:
		return element_codeblock.New(wr, tile)

	default:
		return &elementStub{}
	}
}
