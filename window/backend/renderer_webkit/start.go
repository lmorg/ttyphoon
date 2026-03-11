package rendererwebkit

import (
	"github.com/lmorg/ttyphoon/app"
	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/window/backend/typeface"
)

var currentRenderer *webkitRender

func Initialise() (types.Renderer, *types.XY) {
	glyphSize := calculateGlyphSize()

	r := &webkitRender{
		glyphSize:    glyphSize,
		windowCells:  &types.XY{X: 120, Y: 40},
		windowTitle:  app.Name,
		keyboardMode: types.KeysNormal,
	}

	currentRenderer = r

	return r, r.windowCells
}

func CurrentRenderer() (*webkitRender, bool) {
	if currentRenderer == nil {
		return nil, false
	}

	return currentRenderer, true
}

func GetConfiguredGlyphSize() *types.XY {
	return calculateGlyphSize()
}

func calculateGlyphSize() *types.XY {
	size, err := typeface.MeasureSize(config.Config.TypeFace.FontName, config.Config.TypeFace.FontSize)
	if err != nil {
		panic(err)
	}

	if size == nil || size.X <= 0 || size.Y <= 0 {
		panic("invalid glyph size from typography measurement")
	}

	size.X += int32(config.Config.TypeFace.AdjustCellWidth)
	size.Y += int32(config.Config.TypeFace.AdjustCellHeight)
	return size
}
