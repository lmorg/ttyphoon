package types

type AppWindowTerms struct {
	Tabs   []Tab
	Tiles  []Tile
	Active Tile
}

type Tab interface {
	Name() string
	Id() string
	Index() int
	Active() bool
}

type Tile interface {
	Name() string
	Id() string
	Left() int32
	Top() int32
	Right() int32
	Bottom() int32
	GetTerm() Term
	SetTerm(Term)
}
