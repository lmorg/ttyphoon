package rendersdl

import (
	"github.com/lmorg/mxtty/types"
	elementCsv "github.com/lmorg/mxtty/window/backend/renderer_sdl/element_csv"
	elementImage "github.com/lmorg/mxtty/window/backend/renderer_sdl/element_image"
)

func (sr *sdlRender) NewElement(tileId types.TileId, id types.ElementID) types.Element {
	switch id {
	case types.ELEMENT_ID_IMAGE:
		return elementImage.New(sr, tileId, sr.loadImage)

	case types.ELEMENT_ID_CSV:
		return elementCsv.New(sr, tileId)

	default:
		return nil
	}
}
