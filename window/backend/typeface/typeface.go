package typeface

import (
	"github.com/forPelevin/gomoji"
	"github.com/lmorg/mxtty/config"
	"github.com/lmorg/mxtty/types"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

type typefaceRenderer interface {
	Init() error
	Open(string, int) error
	GetSize() *types.XY
	SetStyle(types.SgrFlag)
	RenderGlyphs(*types.Colour, *sdl.Rect, ...rune) (*sdl.Surface, error)
	glyphIsProvided(int, rune) bool
	Deprecated_GetFont() *ttf.Font
	Close()
}

var renderer typefaceRenderer

func Init() error {
	if config.Config.TypeFace.HarfbuzzRenderer {
		renderer = new(fontHarfbuzz)
	} else {
		renderer = new(fontSdl)
	}

	return renderer.Init()
}

func Open(name string, size int) (err error) {
	return renderer.Open(name, size)
}

func GetSize() *types.XY {
	return renderer.GetSize()
}

func SetStyle(style types.SgrFlag) {
	renderer.SetStyle(style)
}

func GlyphIsEmoji(r rune) bool {
	return gomoji.ContainsEmoji(string(r))
}

func RenderGlyphs(fg *types.Colour, cellRect *sdl.Rect, ch ...rune) (*sdl.Surface, error) {
	return renderer.RenderGlyphs(fg, cellRect, ch...)
}

func Close() {
	ttf.Quit()
}

func Deprecated_GetFont() *ttf.Font {
	return renderer.Deprecated_GetFont()
}

/*
func ligSplitSequence(runes []rune) [][]rune {
	var (
		seq [][]rune
		i   int
	)

	for _, r := range runes {
		if renderer.glyphIsProvided(0, r) {
			seq[i] = append(seq[i], r)
			continue
		}

		if len(seq[i]) == 0 {
			seq[i] = append(seq[i], r)
			seq = append(seq, []rune{})
			i++
		} else {
			seq = append(seq, []rune{r}, []rune{})
			i += 2
		}
	}

	return seq
}
*/
