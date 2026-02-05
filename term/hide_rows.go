package virtualterm

import (
	"errors"
	"fmt"
	"strings"

	"github.com/lmorg/ttyphoon/debug"
	"github.com/lmorg/ttyphoon/types"
)

func (term *Term) HideRows(absStart, absEnd int) error {
	if term.IsAltBuf() {
		return errors.New("this feature is not supported in alt buffer")
	}

	term._mutex.Lock()
	defer term._mutex.Unlock()

	if absStart < 1 {
		absStart = 1
	}

	newBuf := term._scrollBuf
	newBuf = append(newBuf, term._normBuf...)

	if len(newBuf[absStart-1].Hidden) != 0 {
		return errors.New("this row already contains hidden rows")
	}

	newBuf[absStart-1].Hidden = clone(newBuf[absStart:absEnd])
	debug.Log(newBuf[absStart-1].Hidden.String())
	length := len(newBuf[absStart-1].Hidden)
	newBuf = append(newBuf[:absStart], newBuf[absEnd:]...)

	if len(newBuf) < int(term.size.Y) {
		newBuf = append(term.makeScreen(), newBuf...)
	}

	if term._scrollOffset > 0 {
		term._scrollOffset -= absEnd - absStart
	}
	term.updateScrollback()

	term._normBuf = clone(newBuf[len(newBuf)-int(term.size.Y):])
	term._scrollBuf = clone(newBuf[:len(newBuf)-int(term.size.Y)])

	term.renderer.DisplayNotification(types.NOTIFY_INFO, fmt.Sprintf("%d rows have been hidden", length))

	return nil
}

func (term *Term) UnhideRows(absPos int) error {
	if term.IsAltBuf() {
		return errors.New("this feature is not supported in alt buffer")
	}

	var row *types.Row

	if absPos < len(term._scrollBuf) {
		row = term._scrollBuf[absPos]
	} else {
		row = term._normBuf[absPos-len(term._scrollBuf)]
	}

	term.insertRows(absPos, row.Hidden)

	length := len(row.Hidden)
	row.Hidden = nil
	term.renderer.DisplayNotification(types.NOTIFY_INFO, fmt.Sprintf("%d rows have been unhidden", length))

	return nil
}

func (term *Term) insertRowsAtRowId(id uint64, rows types.Screen) error {
	term._mutex.Lock()
	defer term._mutex.Unlock()

	for i := range term._normBuf {
		if term._normBuf[i].Id == id {
			return term._insertRows(i+len(term._scrollBuf), rows)
		}
	}

	for i := range term._scrollBuf {
		if term._scrollBuf[i].Id == id {
			return term._insertRows(i, rows)
		}
	}

	return fmt.Errorf("cannot insert rows: cannot find row with ID %d", id)
}

func (term *Term) insertRows(absPos int, rows types.Screen) error {
	if term.IsAltBuf() {
		return errors.New("this feature is not supported in alt buffer")
	}

	term._mutex.Lock()

	defer term._mutex.Unlock()
	return term._insertRows(absPos, rows)
}

func (term *Term) _insertRows(absPos int, rows types.Screen) error {
	debug.Log(rows.String())

	tmp := term._scrollBuf
	tmp = append(tmp, term._normBuf...)

	newBuf := append(clone(tmp[:absPos+1]), rows...)
	newBuf = clone(append(newBuf, tmp[absPos+1:]...))

	var l int
	for i := len(term._normBuf) - 1; i > 0; i-- {
		if strings.TrimSpace(term._normBuf[i].String()) != "" {
			break
		}
		l++
	}
	l = min(l, len(term._scrollBuf))
	if l > 0 && int(term.curPos().Y) > absPos-len(term._scrollBuf) {
		term._curPos.Y += int32(l)
	}

	term._normBuf = clone(newBuf[len(newBuf)-int(term.size.Y)-l:])
	term._scrollBuf = clone(newBuf[:len(newBuf)-int(term.size.Y)-l])

	return nil
}

func (term *Term) FoldAtIndent(pos *types.XY) error {
	if term.IsAltBuf() {
		return errors.New("folding is not supported in alt buffer")
	}

	row := term.visibleScreen()[pos.Y]
	screen := append(term._scrollBuf, term._normBuf...)
	absPos := term.convertRelPosToAbsPos(pos)

	for i := range row.Cells {
		if row.Cells[i].Rune() != ' ' {
			absPos.X = int32(i)
			absPos.Y++
			_, err := outputBlockFoldIndent(term, screen, absPos, true)
			return err
		}
	}

	return errors.New("cannot fold from an empty line")
}

func outputBlockFoldIndent(term *Term, screen types.Screen, absPos *types.XY, hide bool) (int32, error) {
	x := int(absPos.X)
	var y int32
	for y = absPos.Y + 1; int(y) < len(screen); y++ {
		if screen[y].RowMeta.Is(types.META_ROW_END_BLOCK) {
			break
		}

		phrase, _ := screen.Phrase(int(y))
		debug.Log(phrase)
		if x >= len(phrase) {
			if strings.TrimSpace(phrase) == "" {
				continue
			}
			break
		}

		if strings.TrimSpace(phrase[:x+1]) == "" {
			continue
		}

		break
	}

	if absPos.Y == y {
		return 0, errors.New("nothing to fold")
	}

	if hide {
		term.HideRows(int(absPos.Y), int(y))
	}
	return y, nil
}
