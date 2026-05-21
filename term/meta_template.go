package virtualterm

import (
	"bytes"
	_ "embed"
	"strings"
	"text/template"

	"github.com/lmorg/ttyphoon/app"
	"github.com/lmorg/ttyphoon/types"
)

//go:embed meta_template.md
var mdMetaTemplate string

func commandMeta(term *Term, absPosY int) {
	screen := append(term._scrollBuf, term._normBuf...)
	rowString, err := screen.Phrase(absPosY)
	if err != nil {
		rowString = screen[absPosY].String()
	}

	data := struct {
		AppName   string
		RowString string
		RowId     uint64
		RowMeta   int
		Source    types.RowSource
		Block     types.BlockMeta
		Output    string
	}{
		AppName:   app.Name(),
		RowString: rowString,
		RowId:     screen[absPosY].Id,
		RowMeta:   int(screen[absPosY].RowMeta),
		Source:    *screen[absPosY].Source,
		Block:     *screen[absPosY].Block,
		Output:    screen.PhraseAll(),
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
