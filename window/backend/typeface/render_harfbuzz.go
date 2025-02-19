package typeface

import (
	"image"
	"log"
	"math"
	"regexp"
	"unsafe"

	"github.com/go-text/render"
	"github.com/go-text/typesetting/di"
	"github.com/go-text/typesetting/font"
	"github.com/go-text/typesetting/fontscan"
	"github.com/go-text/typesetting/shaping"
	"github.com/lmorg/mxtty/assets"
	"github.com/lmorg/mxtty/config"
	"github.com/lmorg/mxtty/debug"
	"github.com/lmorg/mxtty/types"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/ttf"
	"golang.org/x/image/math/fixed"
)

var _FONT_FAMILIES = []string{"monospace", "emoji", "math", "fantasy"}

const (
	_STYLE_NORMAL = 0
	_STYLE_BOLD   = 1 << iota
	_STYLE_ITALIC
	_STYLE_FAINT
)

type fontHarfbuzz struct {
	size  *types.XY
	face  map[int]*font.Face
	style int
	fsize float32
	fmap  *fontscan.FontMap
	sdl   *fontSdl // just needed while I figure out how to do everything in harfbuzz
}

func (f *fontHarfbuzz) Init() error {
	f.face = make(map[int]*font.Face)
	f.fsize = float32(config.Config.TypeFace.FontSize)
	f.fmap = fontscan.NewFontMap(log.Default())

	err := f.fmap.UseSystemFonts("")
	if err != nil {
		return err
	}

	f.fmap.SetQuery(fontscan.Query{Families: _FONT_FAMILIES})

	f.sdl = new(fontSdl)
	return f.sdl.Init()
}

func (f *fontHarfbuzz) Open(name string, size int) (err error) {
	if name != "" {
		_FONT_FAMILIES = append([]string{name}, _FONT_FAMILIES...)
		f.setSize()
		name = f.fmap.FontLocation(f.getFace('W').Font).File
		return f.sdl.Open(name, size)
	}

	f.openAsset(assets.TYPEFACE, _STYLE_NORMAL)
	f.openAsset(assets.TYPEFACE_B, _STYLE_BOLD)
	f.openAsset(assets.TYPEFACE_BI, _STYLE_BOLD|_STYLE_ITALIC)
	f.openAsset(assets.TYPEFACE_I, _STYLE_ITALIC)
	f.openAsset(assets.TYPEFACE_L, _STYLE_FAINT)
	f.openAsset(assets.TYPEFACE_LI, _STYLE_FAINT|_STYLE_ITALIC)

	rx := regexp.MustCompile(`[-.]`)
	fontName := rx.Split(assets.TYPEFACE, 2)
	_FONT_FAMILIES = append(fontName[:1], _FONT_FAMILIES...)

	f.setSize()

	return f.sdl.Open("", size)
}

func (f *fontHarfbuzz) openAsset(name string, style int) {
	var (
		res font.Resource
		err error
	)

	res = assets.Reader(name)
	f.face[style], err = font.ParseTTF(res)
	if err != nil {
		panic(err)
	}

	f.fmap.AddFace(f.face[style], fontscan.Location{}, f.face[style].Describe())
}

func (f *fontHarfbuzz) setSize() {
	var shaper shaping.HarfbuzzShaper
	/*ddpi, hdpi, vdpi, err := sdl.GetDisplayDPI(0)
	if err != nil {
		panic(err)
	}
	debug.Log(fmt.Sprintf("DPIs: ddpi(%f), hdpi(%f), vdpi(%f)", ddpi, hdpi, vdpi))

	ddpi, hdpi, vdpi, err = sdl.GetDisplayDPI(1)
	if err != nil {
		panic(err)
	}
	debug.Log(fmt.Sprintf("DPIs: ddpi(%f), hdpi(%f), vdpi(%f)", ddpi, hdpi, vdpi))

	ddpi, hdpi, vdpi, err = sdl.GetDisplayDPI(2)
	if err != nil {
		panic(err)
	}
	debug.Log(fmt.Sprintf("DPIs: ddpi(%f), hdpi(%f), vdpi(%f)", ddpi, hdpi, vdpi))*/

	ddpi := 96

	size := fixed.Int26_6(int(math.Round((float64(config.Config.TypeFace.FontSize) * float64(ddpi) / 72.0) * 64)))
	input := shaping.Input{
		Size: fixed.Int26_6(size),
		//Size:      1,
		Face:      f.getFace('W'),
		Text:      []rune{'W'},
		RunStart:  0,
		RunEnd:    1,
		Direction: di.DirectionLTR,
	}

	output := shaper.Shape(input)

	f.size = &types.XY{
		X: int32(output.Glyphs[0].Width.Ceil()),
		Y: -int32(output.Glyphs[0].Height.Round()) + 4,
	}

	/*f.size = &types.XY{
		X: int32(float32(output.Glyphs[0].Width) / 96 * 72),
		Y: -int32(float32(output.Glyphs[0].Height) / 96 * 72),
	}*/

	debug.Log(f.size)
	//panic("break")
}

func (f *fontHarfbuzz) GetSize() *types.XY {
	//return f.size
	return f.sdl.size
}

func (f *fontHarfbuzz) SetStyle(style types.SgrFlag) {
	query := fontscan.Query{Families: _FONT_FAMILIES}
	f.style = _STYLE_NORMAL

	if style.Is(types.SGR_BOLD) {
		query.Aspect.Weight = font.WeightBold
		f.style |= _STYLE_BOLD
	}

	if style.Is(types.SGR_ITALIC) {
		query.Aspect.Style = font.StyleItalic
		f.style |= _STYLE_ITALIC
	}

	if style.Is(types.SGR_FAINT) {
		query.Aspect.Weight = font.WeightLight
		f.style |= _STYLE_FAINT
	}

	f.fmap.SetQuery(query)
}

func (f *fontHarfbuzz) getFace(ch rune) *font.Face {
	if f.face[f.style] != nil && f.glyphIsProvided(f.style, ch) {
		return f.face[f.style]
	}

	return f.fmap.ResolveFace(ch)
}

// RenderGlyph should be called from a font atlas
func (f *fontHarfbuzz) RenderGlyph(ch rune, fg *types.Colour, cellRect *sdl.Rect) (*sdl.Surface, error) {
	img := image.NewNRGBA(image.Rect(0, 0, int(cellRect.W), int(cellRect.H)))

	textRenderer := &render.Renderer{
		FontSize: f.fsize,
		Color:    fg,
	}

	_ = textRenderer.DrawString(string(ch), img, f.getFace(ch))

	return sdl.CreateRGBSurfaceWithFormatFrom(
		unsafe.Pointer(&img.Pix[0]),
		cellRect.W, cellRect.H,
		32, cellRect.W*4, uint32(sdl.PIXELFORMAT_RGBA32),
	)
}

func (f *fontHarfbuzz) glyphIsProvided(_ int, r rune) bool {
	_, found := f.face[f.style].NominalGlyph(r)
	return found
}

func (f *fontHarfbuzz) Close() {
	f.sdl.Close()
}

func (f *fontHarfbuzz) Deprecated_GetFont() *ttf.Font {
	return f.sdl.Deprecated_GetFont()
}
