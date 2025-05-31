package virtualterm

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/lmorg/mxtty/config"
	"github.com/lmorg/mxtty/types"
	"github.com/lmorg/mxtty/utils/runewidth"
)

func (term *Term) phraseAppend(r rune) {
	if term.IsAltBuf() {
		return
	}

	*term._rowPhrase = append(*term._rowPhrase, r)
}

func (term *Term) phraseSetToRowPos() {
	if term.IsAltBuf() {
		return
	}

	term._rowPhrase = (*term.screen)[term.curPos().Y].Phrase
	(*term.screen)[term.curPos().Y].Source = term._rowSource
	(*term.screen)[term.curPos().Y].Block = term._blockMeta
}

var (
	rxUrl  = regexp.MustCompile(`[a-zA-Z]+://[-./_%&?+=#a-zA-Z0-9]+`)
	rxFile = regexp.MustCompile(`(~|)[-./_%&?+=a-zA-Z0-9]+(\.[a-zA-Z0-9]+|/)`)
)

func (term *Term) autoHotlink(row *types.Row) {
	phrase := string(*row.Phrase)
	posUrl := rxUrl.FindStringIndex(phrase)
	if posUrl != nil {
		if posUrl[0] > int(term.size.X) || posUrl[1] > int(term.size.X) {
			goto skipHttp // link too long
		}
		url := phrase[posUrl[0]:posUrl[1]]
		_strLocToCellPos(phrase, posUrl)
		_autoHotlink(term, row, posUrl, url)
	}

skipHttp:

	rx := rxFile

	posFile := rx.FindAllStringIndex(phrase, -1)
	if posFile == nil {
		return
	}

	for i := range posFile {
		if posFile[i][0] > int(term.size.X) || posFile[i][1] > int(term.size.X) {
			break // filename too long
		}

		file := phrase[posFile[i][0]:posFile[i][1]]
		_strLocToCellPos(phrase, posFile[i])

		if file[0] == '~' {
			home, _ := os.UserHomeDir()
			file = fmt.Sprintf("%s/%s", home, file[1:])
		}
		if file[0] != '/' {
			file = fmt.Sprintf("%s/%s", row.Source.Pwd, file)
		}

		if _, err := os.Stat(file); err == nil {
			file = filepath.Clean(file)
			_autoHotlink(term, row, posFile[i], "file://"+file)
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

	length := pos[1] - pos[0]
	for i := range length {
		row.Cells[pos[0]+i].Element = el
		row.Cells[pos[0]+i].Char = types.SetElementXY(&types.XY{X: int32(i), Y: 0})
	}
}

func _strLocToCellPos(s string, pos []int) {
	if pos[0] > 0 {
		pos[0] = runewidth.StringWidth(s[:pos[0]])
	}

	pos[1] = runewidth.StringWidth(s[:pos[1]])
}
