package types

import "image"

// GraphicsRenderer is optional; terminal graphics never fall back to text cells.
type GraphicsRenderer interface {
	LoadGraphicsImage([]byte) (GraphicsImage, error)
}

type GraphicsImage interface {
	DrawGraphics(Tile, GraphicsPlacement)
	Close()
}

type GraphicsPlacement struct {
	Column, Row           int32
	X, Y, Width, Height   int32
	Source                image.Rectangle
	ClipColumns, ClipRows int32
}
