package virtualterm

import (
	"github.com/lmorg/mxtty/config"
	"github.com/lmorg/mxtty/types"
	"github.com/mattn/go-runewidth"
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

	var wide bool

	if r > 128 && el == nil {
		// A bit of a hack, but runewidth would be slow for every character on
		// on predominantly fast scrolling ASCII text.
		// Another kludge is that we are treating zero width runes as one cell
		// wide.
		wide = runewidth.RuneWidth(r) == 2
	}

	if term._insertOrReplace == _STATE_IRM_INSERT {
		if wide {
			term.csiInsertCharacters(2)
		} else {
			term.csiInsertCharacters(1)
		}
	}

	if term._curPos.X >= term.size.X && !term._noAutoLineWrap {
		term._curPos.X = 0
		phrase := term._rowPhrase // a bit of a hack but we want to...
		term.lineFeed()
		term._rowPhrase = phrase // ...retain the same row for _rowPhrase
	}

	cell := term.currentCell()
	cell.Char = r
	cell.Sgr = term.sgr.Copy()
	cell.Element = el

	/*if debug.Enabled {
		debug.Log(_debugWriteCell{term._curPos, string(r)})
	}*/

	if term._insertOrReplace == _STATE_IRM_REPLACE {
		// add to phrase
		if el == nil && term._activeElement == nil {
			if term._phrase == nil {
				term._phrase = new([]rune)
				cell.Phrase = term._phrase
			}
			*term._phrase = append(*term._phrase, r)
			// ^ old code, delete
			term.phraseAppend(r)
			// ^ new code, keep
		}

		if wide {
			cell.Sgr.Bitwise.Set(types.SGR_WIDE_CHAR)
			term._curPos.X += 2
		} else {
			term._curPos.X++
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
	if err == nil {
		return true
	}

	term.renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
	term._activeElement = nil
	return false
}

func (term *Term) appendScrollBuf(n int) {
	if term.IsAltBuf() {
		return
	}

	term._scrollBuf = append(term._scrollBuf, term._normBuf[0:n]...)

	if len(term._scrollBuf) > config.Config.Terminal.ScrollbackHistory {
		term._scrollBuf = term._scrollBuf[len(term._scrollBuf)-config.Config.Terminal.ScrollbackHistory:]
	}

	if term._scrollOffset > 0 {
		term._scrollOffset += n
		term.updateScrollback()
	}
}
