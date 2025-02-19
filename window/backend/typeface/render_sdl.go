package typeface

import (
	"fmt"
	"log"

	"github.com/flopp/go-findfont"
	"github.com/lmorg/mxtty/assets"
	"github.com/lmorg/mxtty/config"
	"github.com/lmorg/mxtty/debug"
	"github.com/lmorg/mxtty/types"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
)

type fontSdl struct {
	size  *types.XY
	fonts [2]*ttf.Font
}

const (
	_FONT_DEFAULT = iota
	_FONT_EMOJI
)

func (f *fontSdl) Init() error { return ttf.Init() }

func (f *fontSdl) Open(name string, size int) (err error) {
	if name != "" {
		err = f.openSystemTtf(name, size)
	}
	if name == "" || err != nil {
		f.fonts[_FONT_DEFAULT], err = f.openCompiledTtf(assets.TYPEFACE, size)
	}

	if err != nil {
		return err
	}

	f.fonts[_FONT_DEFAULT].SetHinting(ttf.HINTING_MONO)

	f.fonts[_FONT_EMOJI], err = f.openCompiledTtf(assets.EMOJI, size)
	if err != nil {
		panic(err)
	}
	f.fonts[_FONT_EMOJI].SetHinting(ttf.HINTING_MONO)

	err = f._getSize()
	return err
}

func (f *fontSdl) _getSize() error {
	x, y, err := f.fonts[_FONT_DEFAULT].SizeUTF8("W")
	f.size = &types.XY{
		X: int32(x + config.Config.TypeFace.AdjustCellWidth),
		Y: int32(y + config.Config.TypeFace.AdjustCellWidth),
	}
	return err
}

func (f *fontSdl) GetSize() *types.XY {
	return f.size
}

func (f *fontSdl) openSystemTtf(name string, size int) error {
	path, err := findfont.Find(name)
	if err != nil {
		log.Printf("error in findfont.Find(): %s", err.Error())
		log.Println("defaulting to compiled font...")
	}

	f.fonts[_FONT_DEFAULT], err = ttf.OpenFont(path, size)
	if err != nil {
		return fmt.Errorf("error in ttf.OpenFont(): %s", err.Error())
	}

	return nil
}

func (f *fontSdl) openCompiledTtf(assetName string, size int) (*ttf.Font, error) {
	rwops, err := sdl.RWFromMem(assets.Get(assetName))
	if err != nil {
		return nil, fmt.Errorf("error in sdl.RWFromMem(): %s", err.Error())
	}

	font, err := ttf.OpenFontRW(rwops, 0, size)
	if err != nil {
		return nil, fmt.Errorf("error in ttf.OpenFontRW(): %s", err.Error())
	}
	return font, nil
}

func (f *fontSdl) SetStyle(style types.SgrFlag) {
	var ttfStyle ttf.Style

	switch {
	case style.Is(types.SGR_BOLD):
		ttfStyle |= ttf.STYLE_BOLD

	case style.Is(types.SGR_ITALIC):
		ttfStyle |= ttf.STYLE_ITALIC
	}

	f.fonts[_FONT_DEFAULT].SetStyle(ttfStyle)
}

// RenderGlyph should be called from a font atlas
// TODO: support multiple glyphs
func (f *fontSdl) RenderGlyphs(fg *types.Colour, cellRect *sdl.Rect, ch ...rune) (*sdl.Surface, error) {
	for fontId := range f.fonts {
		if f.glyphIsProvided(fontId, ch[0]) {
			return f.fonts[fontId].RenderGlyphBlended(ch[0], sdl.Color{
				R: fg.Red,
				G: fg.Green,
				B: fg.Blue,
				A: fg.Alpha,
			})
		}
		debug.Log(fmt.Sprintf("[sdl] emoji not in font %d: %s", fontId, string(ch)))
	}

	// not found
	return f.fonts[_FONT_DEFAULT].RenderGlyphBlended(ch[0], sdl.Color{
		R: fg.Red,
		G: fg.Green,
		B: fg.Blue,
		A: fg.Alpha,
	})
}

func (f *fontSdl) glyphIsProvided(fontId int, r rune) bool {
	return f.fonts[fontId].GlyphIsProvided(uint16(r))
}

func (f *fontSdl) Close() {
	for _, font := range f.fonts {
		font.Close()
	}
	ttf.Quit()
}

func (f *fontSdl) Deprecated_GetFont() *ttf.Font {
	return f.fonts[_FONT_DEFAULT]
}
