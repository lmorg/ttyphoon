package rendererwebkit

import (
	"bytes"
	"encoding/base64"
	"image"
	_ "image/jpeg"
	_ "image/png"

	"golang.org/x/image/bmp"

	"github.com/lmorg/ttyphoon/types"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type cachedImage struct {
	wr         *webkitRender
	id         int64
	dataURL    string
	sourceSize *types.XY
}

func (wr *webkitRender) loadImage(raw []byte, size *types.XY) (types.Image, error) {
	cfg, format, err := image.DecodeConfig(bytes.NewReader(raw))
	if err != nil {
		cfg, err = bmp.DecodeConfig(bytes.NewReader(raw))
		if err != nil {
			return nil, err
		}
		format = "bmp"
	}

	sourceW := int32(cfg.Width)
	sourceH := int32(cfg.Height)
	if sourceW <= 0 || sourceH <= 0 {
		return nil, image.ErrFormat
	}

	dataURL := "data:image/" + format + ";base64," + base64.StdEncoding.EncodeToString(raw)

	glyphW := float64(1)
	glyphH := float64(1)
	if wr.glyphSize != nil {
		if wr.glyphSize.X > 0 {
			glyphW = float64(wr.glyphSize.X)
		}
		if wr.glyphSize.Y > 0 {
			glyphH = float64(wr.glyphSize.Y)
		}
	}

	if size.X == 0 {
		heightPx := float64(size.Y) * glyphH
		widthPx := (float64(sourceW) / float64(sourceH)) * heightPx
		size.X = int32((widthPx / glyphW) + 1)
		if size.X < 1 {
			size.X = 1
		}
	}

	if wr.windowCells != nil && wr.windowCells.X > 0 && size.X > wr.windowCells.X {
		size.X = wr.windowCells.X
		widthPx := float64(size.X) * glyphW
		heightPx := (float64(sourceH) / float64(sourceW)) * widthPx
		size.Y = int32((heightPx / glyphH) + 1)
		if size.Y < 1 {
			size.Y = 1
		}
	}

	imageID := wr.nextImageID.Add(1)
	if wr.wapp != nil {
		runtime.EventsEmit(wr.wapp, "terminalImageCachePut", map[string]any{
			"id":   imageID,
			"data": dataURL,
		})
	}

	return &cachedImage{
		wr:      wr,
		id:      imageID,
		dataURL: dataURL,
		sourceSize: &types.XY{
			X: sourceW,
			Y: sourceH,
		},
	}, nil
}

func (img *cachedImage) Size() *types.XY {
	return img.sourceSize
}

func (img *cachedImage) Asset() any {
	return img.dataURL
}

func (img *cachedImage) Draw(tile types.Tile, size *types.XY, pos *types.XY) {
	if tile == nil || tile.GetTerm() == nil || size == nil || pos == nil {
		return
	}
	if size.X <= 0 || size.Y <= 0 {
		return
	}

	termSize := tile.GetTerm().GetSize()
	sizeX := size.X
	pcntX := 1.0
	if size.X+pos.X > termSize.X {
		sizeX = termSize.X - pos.X
		pcntX = float64(sizeX) / float64(size.X)
	}

	sizeY := size.Y
	pcntY := 1.0
	if size.Y+pos.Y > termSize.Y {
		sizeY = termSize.Y - pos.Y
		pcntY = float64(sizeY) / float64(size.Y)
	}

	if sizeX <= 0 || sizeY <= 0 {
		return
	}

	srcW := int32(float64(img.sourceSize.X) * pcntX)
	srcH := int32(float64(img.sourceSize.Y) * pcntY)
	if srcW <= 0 || srcH <= 0 {
		return
	}

	img.wr.enqueueDrawCommand(DrawCommand{
		Op:        DrawOpImage,
		X:         tile.Left() + pos.X + 1,
		Y:         tile.Top() + pos.Y,
		Width:     sizeX,
		Height:    sizeY,
		ImageID:   img.id,
		SrcWidth:  srcW,
		SrcHeight: srcH,
		SrcScaleX: pcntX,
		SrcScaleY: pcntY,
	})
}

func (img *cachedImage) Close() {
	if img.wr == nil || img.wr.wapp == nil {
		return
	}

	runtime.EventsEmit(img.wr.wapp, "terminalImageCacheDelete", img.id)
}
