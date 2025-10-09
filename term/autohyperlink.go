package virtualterm

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/lmorg/mxtty/config"
	"github.com/lmorg/mxtty/debug"
	"github.com/lmorg/mxtty/types"
	"github.com/lmorg/mxtty/utils/runewidth"
)

var (
	rxUrl     = regexp.MustCompile(`[a-zA-Z]+://[-./_%&?+=#a-zA-Z0-9]+`)
	rxFile    = regexp.MustCompile(`(~|)[-:./_%&?+=\pL\pN\pP]+`)
	rxSrcLine = regexp.MustCompile(`:[0-9]+$`)
	//rxFile    = regexp.MustCompile(`(~|)[-:./_%&?+=a-zA-Z0-9]+`)
)

func (term *Term) autoHyperlink(rows types.Screen) {
	if !config.Config.Terminal.AutoHyperlink {
		return
	}

	if rows[0].RowMeta.Is(types.META_ROW_AUTO_HYPERLINKED) {
		return
	}

	phrase, _ := rows.Phrase(0)

	_autoHyperlinkUrls(term, rows, phrase)
	_autoHyperlinkFiles(term, rows, phrase)

	for _, row := range rows {
		row.RowMeta.Set(types.META_ROW_AUTO_HYPERLINKED)
	}
}

func _autoHyperlinkUrls(term *Term, rows []*types.Row, phrase string) {
	posUrl := rxUrl.FindAllStringIndex(phrase, -1)
	if posUrl == nil {
		return
	}

	for i := range posUrl {
		url := phrase[posUrl[i][0]:posUrl[i][1]]
		_autoHyperlinkElement(term, rows, phrase, posUrl[i], url, url)
	}
}

func _autoHyperlinkFiles(term *Term, rows []*types.Row, phrase string) {
	posFile := rxFile.FindAllStringIndex(phrase, -1)
	if posFile == nil {
		return
	}

	for i := range posFile {
		label := phrase[posFile[i][0]:posFile[i][1]]
		file := rxSrcLine.ReplaceAllString(label, "")

		if len(file)==0 {
			continue
		}

		if file[0] == '~' {
			home, _ := os.UserHomeDir()
			file = fmt.Sprintf("%s/%s", home, file[1:])
		}
		if file[0] != '/' {
			var pwd string
			if rows[0].Source != nil {
				pwd = rows[0].Source.Pwd
			} else {
				pwd, _ = os.Getwd()
			}
			file = fmt.Sprintf("%s/%s", pwd, file)
		}

		if _, err := os.Stat(file); err == nil {
			file = filepath.Clean(file)
			_autoHyperlinkElement(term, rows, phrase, posFile[i], label, "file://"+file)
		}
	}
}

func _autoHyperlinkElement(term *Term, rows []*types.Row, phrase string, pos []int, label, link string) {
	defer func() {
		if r := recover(); r != nil {
			debug.Log(r)
		}
	}()
	debug.Log(label)

	acp := types.NewApcSliceNoParse([]string{label, link})
	el := term.renderer.NewElement(term.tile, types.ELEMENT_ID_HYPERLINK)
	err := el.Generate(acp)
	if err != nil {
		return
	}

	startCell := runewidth.StringWidth(string(phrase[:pos[0]]))
	endCell := runewidth.StringWidth(string(phrase[pos[0]:pos[1]])) + startCell
	rLabel, r := []rune(label), 0
	termSizeX := int(term.size.X)

	x, y, z := startCell, int32(0), int32(0)
	var wide bool
	for i := startCell; i < endCell; i++ {
		if x >= termSizeX {
			y++
			x = 0
		}

		rows[y].Cells[x].Element = el
		rows[y].Cells[x].Char = types.SetElementXY(&types.XY{X: z, Y: 0})
		if wide {
			wide = false
			r--
		} else if runewidth.RuneWidth(rLabel[r]) == 2 {
			rows[y].Cells[x].Sgr.Bitwise.Set(types.SGR_WIDE_CHAR)
			wide = true
		}
		x++
		z++
		r++
	}
}
