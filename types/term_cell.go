package types

type Cell struct {
	Char    rune
	Sgr     *Sgr
	Element Element
	//Phrase  *[]rune
}

func (c *Cell) Clear() {
	c.Char = 0
	c.Sgr = &Sgr{}
	c.Element = nil
}

func (c *Cell) HasTextElement() bool {
	_, ok := c.Element.(*TextElement)
	return ok
}

func (c *Cell) HasDrawableElement() bool {
	return c.Element != nil && !c.HasTextElement()
}

func (c *Cell) Rune() rune {
	switch {
	case c.HasDrawableElement():
		return c.Element.Rune(c.GetElementXY())

	case c.Char == 0:
		return ' '

	default:
		return c.Char
	}
}

const (
	_CELL_ELEMENTXY_MASK    = (^int32(0)) << 16
	_CELL_ELEMENTXY_CEILING = int32(^uint16(0) >> 1)
)

func SetElementXY(xy *XY) rune {
	if xy.X > _CELL_ELEMENTXY_CEILING || xy.Y > _CELL_ELEMENTXY_CEILING {
		panic("TODO: proper error handling")
	}
	return (xy.X << 16) | xy.Y
}

func (c *Cell) GetElementXY() *XY {
	if c.HasTextElement() {
		return &XY{}
	}
	return &XY{
		X: c.Char >> 16,
		Y: c.Char &^ _CELL_ELEMENTXY_MASK,
	}
}
