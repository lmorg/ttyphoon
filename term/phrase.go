package virtualterm

import (
	"fmt"
	"os"
	"regexp"

	"github.com/lmorg/mxtty/config"
	"github.com/lmorg/mxtty/types"
)

func (term *Term) phraseAppend(r rune) {
	if term.IsAltBuf() {
		return
	}

	*term._rowPhrase = append(*term._rowPhrase, r)
}

/*func (term *Term) phraseErasePhrase() {
	if term.IsAltBuf() {
		return
	}

	term._rowPhrase = &[]rune{}
}*/

func (term *Term) phraseSetToRowPos() {
	if term.IsAltBuf() {
		return
	}

	term._rowPhrase = (*term.screen)[term.curPos().Y].Phrase
}

var (
	rxUrl  = regexp.MustCompile(`http(|s)://[-./_%&?+=a-zA-Z0-9]+`)
	rxFile = regexp.MustCompile(`[-./_%&?+=a-zA-Z0-9]+\.[a-zA-Z0-9]+`)
)

func (term *Term) autoHotlink(row *types.Row) {
	phrase := string(*row.Phrase)
	posUrl := rxUrl.FindStringIndex(phrase)
	if posUrl != nil {
		_autoHotlink(term, row, posUrl, "")
	}

	posFile := rxFile.FindAllStringIndex(phrase, -1)
	if posFile == nil {
		return
	}

	for i := range posFile {
		file := phrase[posFile[i][0]:posFile[i][1]]
		if file[0] != '/' {
			file = fmt.Sprintf("%s/%s", term.tile.Path(), file)
		}
		if _, err := os.Stat(file); err == nil {
			_autoHotlink(term, row, posFile[i], file)
		}
	}
}

func _autoHotlink(term *Term, row *types.Row, pos []int, path string) {
	if !config.Config.Terminal.AutoHotlink {
		return
	}

	display := string((*row.Phrase)[pos[0]:pos[1]])
	if path == "" {
		path = display
	}

	acp := types.NewApcSliceNoParse([]string{path, display})
	el := term.renderer.NewElement(term.tile, types.ELEMENT_ID_HYPERLINK)
	err := el.Generate(acp, row.Cells[pos[0]].Sgr)
	if err != nil {
		return
	}

	length := pos[1] - pos[0] - 1
	for i := range length {
		row.Cells[pos[0]+i].Element = el
		row.Cells[pos[0]+i].Char = types.SetElementXY(&types.XY{int32(i), 0})
	}
}
