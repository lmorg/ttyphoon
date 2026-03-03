package virtualterm

import (
	"bytes"
	_ "embed"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/lmorg/ttyphoon/types"
)

//go:embed debug_template.md
var mdDebugTemplate string

func notesDebug(term *Term, absPosY int) {
	screen := append(term._scrollBuf, term._normBuf...)

	data := struct {
		RowString string
		RowId     uint64
		RowMeta   int
		Source    types.RowSource
		Block     types.BlockMeta
	}{
		RowString: screen[absPosY].String(),
		RowId:     screen[absPosY].Id,
		RowMeta:   int(screen[absPosY].RowMeta),
		Source:    *screen[absPosY].Source,
		Block:     *screen[absPosY].Block,
	}

	tmpl, err := template.New("cmd").Funcs(debugTemplateFuncs()).Parse(mdDebugTemplate)
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

	filename := fmt.Sprintf("debug-%d.md", time.Now().Unix())
	term.renderer.NotesCreateAndOpen(filename, buf.String())
}

func debugTemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"toString": func(r []rune) string { return string(r) },
		"quote":    func(s string) string { return strings.ReplaceAll(s, "\n", "\n> ") },
	}
}
