package rendersdl

import (
	"log"

	"github.com/lmorg/mxtty/types"
)

func (sr *sdlRender) DrawTable(tileId types.TileId, _pos *types.XY, height int32, boundaries []int32) {
	var err error

	pos := types.XY{
		X: _pos.X + sr.termWin.Tiles[tileId].TopLeft.X,
		Y: _pos.Y + sr.termWin.Tiles[tileId].TopLeft.Y,
	}

	fg := types.SGR_DEFAULT.Fg

	texture := sr.createRendererTexture()
	if texture == nil {
		return
	}
	defer sr.restoreRendererTexture()

	sr.renderer.SetDrawColor(fg.Red, fg.Green, fg.Blue, 128)

	X := (pos.X * sr.glyphSize.X) + _PANE_LEFT_MARGIN
	Y := (pos.Y * sr.glyphSize.Y) + _PANE_TOP_MARGIN
	H := Y + ((height + 1) * sr.glyphSize.Y) //- 1

	err = sr.renderer.DrawLine(X, Y, X, H)
	if err != nil {
		log.Printf("ERROR: %s", err.Error())
		return
	}

	for i := range boundaries {
		x := X + (boundaries[i] * sr.glyphSize.X) //- 1
		err = sr.renderer.DrawLine(x, Y, x, H)
		if err != nil {
			log.Printf("ERROR: %s", err.Error())
			return
		}
	}

	x := X + (boundaries[len(boundaries)-1] * sr.glyphSize.X) //- 1
	y := Y + ((height + 1) * sr.glyphSize.Y)                  //- 1
	err = sr.renderer.DrawLine(X, y, x, y)
	if err != nil {
		log.Printf("ERROR: %s", err.Error())
		return
	}

	sr.renderer.SetDrawColor(fg.Red, fg.Green, fg.Blue, 100)

	for i := int32(0); i <= height; i++ {
		y = Y + (i * sr.glyphSize.Y) //- 1
		err = sr.renderer.DrawLine(X, y, x, y)
		if err != nil {
			log.Printf("ERROR: %s", err.Error())
			return
		}
	}
}
