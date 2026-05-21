package virtualterm

import (
	"bytes"
	_ "embed"
	"fmt"
	"strings"
	"text/template"

	"github.com/lmorg/ttyphoon/app"
	"github.com/lmorg/ttyphoon/types"
)

//go:embed meta_template.md
var mdMetaTemplate string

const (
	_FMT_META_DATE = "2006-01-02"
	_FMT_META_TIME = "15:04:05"
)

func commandMeta(term *Term, absPosY int) {
	screen := append(term._scrollBuf, term._normBuf...)
	rowString, err := screen.Phrase(absPosY)
	if err != nil {
		rowString = screen[absPosY].String()
	}

	outputBlock := term.copyOutputBlock(term.getBlockStartAndEndAbs(absPosY))
	durationMilli := screen[absPosY].Block.TimeEnd.UnixMilli() - screen[absPosY].Block.TimeStart.UnixMilli()

	data := struct {
		AppName   string
		RowString string
		RowId     uint64
		RowMeta   int
		Source    types.RowSource
		Block     types.BlockMeta
		Output    string
		DateStart string
		TimeStart string
		TimeEnd   string
		Duration  string
	}{
		AppName:   app.Name(),
		RowString: rowString,
		RowId:     screen[absPosY].Id,
		RowMeta:   int(screen[absPosY].RowMeta),
		Source:    *screen[absPosY].Source,
		Block:     *screen[absPosY].Block,
		Output:    string(outputBlock),
		DateStart: screen[absPosY].Block.TimeStart.Format(_FMT_META_DATE),
		TimeStart: screen[absPosY].Block.TimeStart.Format(_FMT_META_TIME),
		TimeEnd:   screen[absPosY].Block.TimeEnd.Format(_FMT_META_TIME),
		Duration:  fmt.Sprintf("%d ms", durationMilli),
	}

	tmpl, err := template.New("cmd").Funcs(metaTemplateFuncs()).Parse(mdMetaTemplate)
	if err != nil {
		term.renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
		return
	}

	var b []byte
	buf := bytes.NewBuffer(b)
	err = tmpl.Execute(buf, data)
	if err != nil {
		term.renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
	}

	term.renderer.DisplayMarkdownModel(buf.String())

	//filename := fmt.Sprintf("debug-%d.md", time.Now().Unix())
	//term.renderer.NotesCreateAndOpen(filename, buf.String())
}

func metaTemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"toString": func(r []rune) string { return string(r) },
		"quote":    func(s string) string { return strings.ReplaceAll(s, "\n", "\n> ") },
		"trim":     func(s string) string { return strings.TrimSpace(s) },
	}
}
