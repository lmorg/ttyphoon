package rendererwebkit

import (
	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/window/elements/element_codeblock"
	"github.com/lmorg/ttyphoon/window/elements/element_hyperlink"
	"github.com/lmorg/ttyphoon/window/elements/element_image"
	"github.com/lmorg/ttyphoon/window/elements/element_table"
)

func (wr *webkitRender) NewElement(tile types.Tile, id types.ElementID) types.Element {
	switch id {
	case types.ELEMENT_ID_IMAGE:
		return element_image.NewBitmap(wr, tile, wr.loadImage)

	case types.ELEMENT_ID_SIXEL:
		return element_image.NewSixel(wr, tile, wr.loadImage)

	case types.ELEMENT_ID_CSV:
		return element_table.NewCsv(wr, tile)

	case types.ELEMENT_ID_MARKDOWN_TABLE:
		return element_table.NewMarkdown(wr, tile)

	case types.ELEMENT_ID_HYPERLINK:
		return element_hyperlink.New(wr, tile)

	case types.ELEMENT_ID_CODEBLOCK:
		return element_codeblock.New(wr, tile)

	default:
		return &elementStub{}
	}
}

type elementStub struct{}

func (es *elementStub) Generate(_ *types.ApcSlice) error { return nil }
func (es *elementStub) Write(_ rune) error               { return nil }
func (es *elementStub) Rune(_ *types.XY) rune            { return 0 }
func (es *elementStub) Size() *types.XY                  { return &types.XY{} }
func (es *elementStub) Draw(_ *types.XY)                 {}
func (es *elementStub) MouseClick(_ *types.XY, _ types.MouseButtonT, _ uint8, _ types.ButtonStateT, _ types.EventIgnoredCallback) {
}
func (es *elementStub) MouseWheel(_ *types.XY, _ *types.XY, _ types.EventIgnoredCallback) {}
func (es *elementStub) MouseMotion(_ *types.XY, _ *types.XY, _ types.EventIgnoredCallback) {
}
func (es *elementStub) MouseHover(_ *types.XY, _ *types.XY) func() { return func() {} }
func (es *elementStub) MouseOut()                                  {}
