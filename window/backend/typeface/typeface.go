package typeface

import (
	"github.com/forPelevin/gomoji"
	"github.com/lmorg/mxtty/config"
	"github.com/lmorg/mxtty/types"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

var harfbuzz = new(fontHarfbuzz)

func Init() {
	harfbuzz = new(fontHarfbuzz)
	err := harfbuzz.Init()
	if err != nil {
		panic(err)
	}

	err = ttf.Init()
	if err != nil {
		panic(err)
	}

	err = harfbuzz.Open(
		config.Config.TypeFace.FontName,
		config.Config.TypeFace.FontSize,
	)
	if err != nil {
		panic(err.Error())
	}
}

func GetSize() *types.XY {
	return harfbuzz.getSize()
}

func SetStyle(style types.SgrFlag) {
	harfbuzz.SetStyle(style)
}

func GlyphIsEmoji(r rune) bool {
	return gomoji.ContainsEmoji(string(r))
}

func RenderGlyphs(fg *types.Colour, cellRect *sdl.Rect, ch ...rune) (*sdl.Surface, error) {
	return harfbuzz.RenderGlyphs(fg, cellRect, ch...)
}

func Close() {
	ttf.Quit()
}
