package rendersdl

import (
	"github.com/lmorg/ttyphoon/types"
)

func (sr *sdlRender) NewElement(tile types.Tile, id types.ElementID) types.Element {
	/*switch id {
	case types.ELEMENT_ID_IMAGE:
		return element_image.NewBitmap(sr, tile, sr.loadImage)

	case types.ELEMENT_ID_SIXEL:
		return element_image.NewSixel(sr, tile, sr.loadImage)

	case types.ELEMENT_ID_CSV:
		return element_table.NewCsv(sr, tile)

	case types.ELEMENT_ID_MARKDOWN_TABLE:
		return element_table.NewMarkdown(sr, tile)

	case types.ELEMENT_ID_HYPERLINK:
		return element_hyperlink.New(sr, tile)

	case types.ELEMENT_ID_CODEBLOCK:
		return element_codeblock.New(sr, tile)*/

	//default:
	return nil
	//}
}
