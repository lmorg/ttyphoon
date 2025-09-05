package virtualterm

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/lmorg/mxtty/config"
	"github.com/lmorg/mxtty/debug"
	"github.com/lmorg/mxtty/types"
)

func (term *Term) phraseSetToRowPos(flags linefeedF) {
	/*if term.IsAltBuf() {
		return
	}*/

	if flags.Is(_LINEFEED_LINE_OVERFLOWED) {
		(*term.screen)[term.curPos().Y].RowMeta.Set(types.META_ROW_FROM_LINE_OVERFLOW)
	} else {
		(*term.screen)[term.curPos().Y].RowMeta.Unset(types.META_ROW_FROM_LINE_OVERFLOW)
	}

	(*term.screen)[term.curPos().Y].Source = term._rowSource
	(*term.screen)[term.curPos().Y].Block = term._blockMeta
}

var (
	rxUrl  = regexp.MustCompile(`[a-zA-Z]+://[-./_%&?+=#a-zA-Z0-9]+`)
	rxFile = regexp.MustCompile(`(~|)[-:./_%&?+=a-zA-Z0-9]+(\.[a-zA-Z0-9]+|/)`)
)

/*func (term *Term) autoHotlink(row int, phrase string) {
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

	display := row.String()[pos[0]:pos[1]]
	if path == "" {
		path = display
	}

	acp := types.NewApcSliceNoParse([]string{path, display})
	el := term.renderer.NewElement(term.tile, types.ELEMENT_ID_HYPERLINK)
	err := el.Generate(acp)
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
*/

func (term *Term) autoHotlink(rows types.Screen) {
	if !config.Config.Terminal.AutoHotlink {
		return
	}

	if rows[0].RowMeta.Is(types.META_ROW_AUTO_HOTLINKED) {
		return
	}

	phrase, _ := rows.Phrase(0)

	_autoHotlinkUrls(term, rows, phrase)
	_autoHotlinkFiles(term, rows, phrase)

	for _, row := range rows {
		row.RowMeta.Set(types.META_ROW_AUTO_HOTLINKED)
	}
}

/*func _strLocToCellPos(s string, pos []int) {
	if pos[0] > 0 {
		pos[0] = runewidth.StringWidth(s[:pos[0]])
	}

	pos[1] = runewidth.StringWidth(s[:pos[1]])
}*/

func _autoHotlinkUrls(term *Term, rows []*types.Row, phrase string) {
	posUrl := rxUrl.FindAllStringIndex(phrase, -1)
	if posUrl == nil {
		return
	}

	for i := range posUrl {
		url := phrase[posUrl[i][0]:posUrl[i][1]]
		//_strLocToCellPos(phrase, posUrl[i])
		_autoHotlinkElement(term, rows, posUrl[i], url, url)
	}
}

func _autoHotlinkFiles(term *Term, rows []*types.Row, phrase string) {
	posFile := rxFile.FindAllStringIndex(phrase, -1)
	if posFile == nil {
		return
	}

	for i := range posFile {
		/*if posFile[i][0] > int(term.size.X) || posFile[i][1] > int(term.size.X) {
			break // filename too long
		}*/

		label := phrase[posFile[i][0]:posFile[i][1]]
		file := label
		//_strLocToCellPos(phrase, posFile[i])

		if file[0] == '~' {
			home, _ := os.UserHomeDir()
			file = fmt.Sprintf("%s/%s", home, file[1:])
		}
		if file[0] != '/' {
			file = fmt.Sprintf("%s/%s", rows[0].Source.Pwd, file)
		}

		if _, err := os.Stat(file); err == nil {
			file = filepath.Clean(file)
			_autoHotlinkElement(term, rows, posFile[i], label, "file://"+file)
		}
	}
}

func _autoHotlinkElement(term *Term, rows []*types.Row, pos []int, label, link string) {
	debug.Log(label)

	acp := types.NewApcSliceNoParse([]string{label, link})
	el := term.renderer.NewElement(term.tile, types.ELEMENT_ID_HYPERLINK)
	err := el.Generate(acp)
	if err != nil {
		return
	}

	x, y, z := int32(pos[0]), int32(0), int32(0)
	for i := pos[0]; i < pos[1]; i++ {
		if x >= term.size.X {
			y++
			x = 0
		}

		rows[y].Cells[x].Element = el
		rows[y].Cells[x].Char = types.SetElementXY(&types.XY{X: z, Y: 0})
		x++
		z++
	}
}
