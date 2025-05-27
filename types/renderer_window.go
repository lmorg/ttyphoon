package types

type AppWindowTerms struct {
	Tabs   []Tab
	Tiles  []Tile
	Active Tile
}

type Tab interface {
	Name() string
	Rename(string) error
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
	AtBottom() bool
	GetTerm() Term
	SetTerm(Term)
	Pwd() string
	Close()
}
