package rendersdl

import (
	"github.com/lmorg/mxtty/types"
	elementCsv "github.com/lmorg/mxtty/window/backend/renderer_sdl/element_csv"
	elementImage "github.com/lmorg/mxtty/window/backend/renderer_sdl/element_image"
)

func (sr *sdlRender) NewElement(tile *types.Tile, id types.ElementID) types.Element {
	switch id {
	case types.ELEMENT_ID_IMAGE:
		return elementImage.New(sr, tile, sr.loadImage)

	case types.ELEMENT_ID_CSV:
		return elementCsv.New(sr, tile)

	default:
		return nil
	}
}
