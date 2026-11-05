package rendererwebkit

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"reflect"
	"testing"

	"github.com/lmorg/ttyphoon/types"
)

type graphicsTestTile struct{ types.Tile }

func (graphicsTestTile) Left() int32 { return 4 }
func (graphicsTestTile) Top() int32  { return 2 }

func TestGraphicsImageCacheAndDrawCommand(t *testing.T) {
	wr := &webkitRender{}
	raw := image.NewNRGBA(image.Rect(0, 0, 2, 1))
	raw.Set(0, 0, color.NRGBA{R: 255, A: 128})
	var data bytes.Buffer
	if err := png.Encode(&data, raw); err != nil {
		t.Fatal(err)
	}
	asset, err := wr.LoadGraphicsImage(data.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	cached := asset.(*cachedImage)
	if cached.sourceSize.X != 2 || cached.sourceSize.Y != 1 {
		t.Fatal("source dimensions changed")
	}
	asset.DrawGraphics(graphicsTestTile{}, types.GraphicsPlacement{
		Column: 3, Row: -1, X: 2, Y: 4, Width: 50, Height: 25,
		Source: image.Rect(1, 0, 2, 1), ClipColumns: 80, ClipRows: 24,
	})
	// The compact command uses cell anchors but pixel-based crop/size/offsets.
	want := []DrawOpTuple{{compactOpGraphicsImage, int32(8), int32(1), int32(2), int32(4),
		int32(50), int32(25), cached.id, int32(1), int32(0), int32(1), int32(1),
		int32(5), int32(2), int32(80), int32(24)}}
	if got := encodeDrawCommands(wr.drawCommands); !reflect.DeepEqual(got, want) {
		t.Fatalf("wrong graphics command: %#v", got)
	}
	asset.Close()
}
