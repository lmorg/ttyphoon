package rendersdl

import (
	"github.com/lmorg/ttyphoon/types"
	"github.com/veandco/go-sdl2/sdl"
)

func newFontSurface(glyphSize *types.XY, nCharacters int32) *sdl.Surface {
	surface, err := sdl.CreateRGBSurfaceWithFormat(0, glyphSize.X*nCharacters, glyphSize.Y*_HLTEXTURE_LAST, 32, uint32(sdl.PIXELFORMAT_RGBA32))
	if err != nil {
		panic(err) // TODO: better error handling please!
	}

	pixel := sdl.MapRGBA(surface.Format, types.SGR_COLOR_BACKGROUND.Red, types.SGR_COLOR_BACKGROUND.Green, types.SGR_COLOR_BACKGROUND.Blue, 255)
	err = surface.FillRect(&sdl.Rect{W: surface.W, H: surface.H}, pixel)
	if err != nil {
		panic(err) // TODO: better error handling please!
	}

	err = surface.SetColorKey(true, pixel)
	if err != nil {
		panic(err) // TODO: better error handling please!
	}

	return surface
}
