package rendersdl

import (
	"github.com/lmorg/mxtty/types"
	elementCsv "github.com/lmorg/mxtty/window/backend/renderer_sdl/element_csv"
	elementHyperlink "github.com/lmorg/mxtty/window/backend/renderer_sdl/element_hyperlink"
	elementImage "github.com/lmorg/mxtty/window/backend/renderer_sdl/element_image"
	elementSixel "github.com/lmorg/mxtty/window/backend/renderer_sdl/element_sixel"
)

func (sr *sdlRender) NewElement(tile types.Tile, id types.ElementID) types.Element {
	switch id {
	case types.ELEMENT_ID_IMAGE:
		return elementImage.New(sr, tile, sr.loadImage)

	case types.ELEMENT_ID_SIXEL:
		return elementSixel.New(sr, tile, sr.loadImage)

	case types.ELEMENT_ID_CSV:
		return elementCsv.New(sr, tile)

	case types.ELEMENT_ID_HYPERLINK:
		return elementHyperlink.New(sr, tile)

	default:
		return nil
	}
}
