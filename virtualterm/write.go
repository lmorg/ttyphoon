package virtualterm

import (
	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/utils/runewidth"
)

type _debugWriteCell struct {
	types.XY
	Rune string
}

func (term *Term) writeCell(r rune, el types.Element) {
	term._mousePosRenderer.Set(nil)

	if term.writeToElement(r) {
		return
	}

	charWidth := 1
	var wide bool

	if r > 128 && el == nil {
		// A bit of a hack, but runewidth would be slow for every character on
		// on predominantly fast scrolling ASCII text.
		charWidth = runewidth.RuneWidth(r)
		wide = charWidth == 2

		// Drop zero-width runes as standalone cells so they don't consume
		// columns and trigger spurious line wrapping.
		if charWidth == 0 {
			return
		}
	}

	if term._insertOrReplace == _STATE_IRM_INSERT {
		term.csiInsertCharacters(int32(charWidth))
	}

	if term._curPos.X >= term.size.X && !term._noAutoLineWrap {
		term._curPos.X = 0
		term.lineFeed(_LINEFEED_LINE_OVERFLOWED)

	}

	cell := term.currentCell()
	cell.Char = r
	cell.Sgr = term.sgr.Copy()
	cell.Element = el

	/*if debug.Enabled {
		debug.Log(_debugWriteCell{term._curPos, string(r)})
	}*/

	if term._insertOrReplace == _STATE_IRM_REPLACE {
		if wide {
			cell.Sgr.Bitwise.Set(types.SGR_WIDE_CHAR)
			term._curPos.X += 2
		} else {
			term._curPos.X += int32(charWidth)
		}
	} else if wide {
		// only run this on insert
		cell.Sgr.Bitwise.Set(types.SGR_WIDE_CHAR)
	}

	if term._ssFrequency == 0 {
		term.renderer.TriggerRedraw()
	}
}

func (term *Term) writeToElement(r rune) (ok bool) {
	if term._activeElement == nil {
		return false
	}

	err := term._activeElement.Write(r)
	if err != nil {
		term.renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
		term._activeElement = nil
		return false
	}

	return true
}

func (term *Term) appendScrollBuf(n int) {
	if term.IsAltBuf() {
		return
	}

	term._scrollBuf = append(term._scrollBuf, term._normBuf[0:n]...)

	if len(term._scrollBuf) > config.Config.Terminal.ScrollbackHistory {
		term.cropScrollBuf(config.Config.Terminal.ScrollbackHistory)
		//term._scrollBuf = term._scrollBuf[len(term._scrollBuf)-config.Config.Terminal.ScrollbackHistory:]
	}

	if term._scrollOffset > 0 {
		term._scrollOffset += n
		term.updateScrollback()
	}
}

func (term *Term) cropScrollBuf(n int) {
	term.historyDb.Append(term._scrollBuf[:n])
	term._scrollBuf = term._scrollBuf[len(term._scrollBuf)-n:]
}
