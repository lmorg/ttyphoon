package rendersdl

import (
	"github.com/lmorg/mxtty/types"
	"github.com/lmorg/mxtty/window/backend/renderer_sdl/layer"
	"github.com/veandco/go-sdl2/sdl"
)

type cachedImage struct {
	surface   *sdl.Surface
	sr        *sdlRender
	sizeCells *types.XY
	rwops     *sdl.RWops
	texture   *sdl.Texture
}

func (sr *sdlRender) loadImage(bmp []byte, size *types.XY) (types.Image, error) {
	rwops, err := sdl.RWFromMem(bmp)
	if err != nil {
		return nil, err
	}

	img := cachedImage{sr: sr, rwops: rwops}

	img.surface, err = sdl.LoadBMPRW(rwops, true)
	if err != nil {
		return nil, err
	}

	img.sizeCells = &types.XY{
		X: sr.glyphSize.X * size.X,
		Y: sr.glyphSize.Y * size.Y,
	}

	if size.X == 0 {
		img.sizeCells.X = int32((float64(img.surface.W) / float64(img.surface.H)) * float64(img.sizeCells.Y))
		size.X = int32((float64(img.sizeCells.X) / float64(sr.glyphSize.X)) + 1)
	}

	wPx, _, err := sr.renderer.GetOutputSize()
	if err != nil {
		return nil, err
	}
	if img.sizeCells.X > wPx {
		img.sizeCells.X = wPx
		img.sizeCells.Y = int32((float64(img.surface.H) / float64(img.surface.W)) * float64(img.sizeCells.X))
		size.X = int32((float64(img.sizeCells.X) / float64(sr.glyphSize.X)))
		size.Y = int32((float64(img.sizeCells.Y) / float64(sr.glyphSize.Y)) + 1)
	}

	img.texture, err = img.sr.renderer.CreateTextureFromSurface(img.surface)
	if err != nil {
		return nil, err
	}

	return &img, nil
}

func (img *cachedImage) Size() *types.XY {
	return img.sizeCells
}

func (img *cachedImage) Draw(size *types.XY, pos *types.XY) {
	srcRect := &sdl.Rect{
		W: img.surface.W,
		H: img.surface.H,
	}

	dstRect := &sdl.Rect{
		X: (pos.X * img.sr.glyphSize.X) + _PANE_LEFT_MARGIN,
		Y: (pos.Y * img.sr.glyphSize.Y) + _PANE_TOP_MARGIN,
		W: size.X * img.sr.glyphSize.X,
		H: size.Y * img.sr.glyphSize.Y,
	}

	img.sr.AddToElementStack(&layer.RenderStackT{img.texture, srcRect, dstRect, false})
}

func (img *cachedImage) Asset() any {
	return img.surface
}

func (img *cachedImage) Close() {
	img.texture.Destroy()
	img.surface.Free()
	img.rwops.Free()
}
