package virtualterm

import (
	"sort"

	"github.com/lmorg/ttyphoon/debug"
)

/*
	TAB STOPS
*/

func (term *Term) c1AddTabStop() {
	debug.Log(term.curPos())

	term._tabStops = append(term._tabStops, term.curPos().X)
	sort.Slice(term._tabStops, func(i, j int) bool { return term._tabStops[i] < term._tabStops[j] })
}

func (term *Term) csiResetTabStops() {
	term._tabStops = make([]int32, 0)
}

func (term *Term) nextTabStop() int32 {
	pos := term.curPos()

	if len(term._tabStops) == 0 {
		return (((pos.X / term._tabWidth) + 1) * term._tabWidth) - pos.X
	}

	for _, tabStop := range term._tabStops {
		if tabStop > pos.X {
			return tabStop - pos.X
		}
	}

	// end of screen
	return term.size.X - pos.X
}

func (term *Term) printTab() {
	term._printTab(term.nextTabStop() + 1)
}

func (term *Term) _printTab(tabWidth int32) {
	for i := 1; i < int(tabWidth); i++ {
		term.writeCell(' ', nil)
	}
}

func (term *Term) csiClearTabStop() {
	pos := term.curPos()
	for i := 0; i < len(term._tabStops); i++ {
		if term._tabStops[i] == pos.X {
			term._tabStops = append(term._tabStops[:i], term._tabStops[i+1:]...)
			i--
		}
	}
}
