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

var (
	rxUrl     = regexp.MustCompile(`[a-zA-Z]+://[-./_%&?+=#a-zA-Z0-9]+`)
	rxFile    = regexp.MustCompile(`(~|)[-:./_%&?+=a-zA-Z0-9]+`)
	rxSrcLine = regexp.MustCompile(`:[0-9]+$`)
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
		_autoHyperlinkElement(term, rows, posUrl[i], url, url)
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

		if file[0] == '~' {
			home, _ := os.UserHomeDir()
			file = fmt.Sprintf("%s/%s", home, file[1:])
		}
		if file[0] != '/' {
			file = fmt.Sprintf("%s/%s", rows[0].Source.Pwd, file)
		}

		if _, err := os.Stat(file); err == nil {
			file = filepath.Clean(file)
			_autoHyperlinkElement(term, rows, posFile[i], label, "file://"+file)
		}
	}
}

func _autoHyperlinkElement(term *Term, rows []*types.Row, pos []int, label, link string) {
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
