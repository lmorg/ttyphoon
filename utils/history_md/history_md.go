package historymd

import (
	_ "embed"
	"fmt"
	"os"
	"text/template"

	"github.com/adrg/xdg"
	"github.com/lmorg/ttyphoon/app"
	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/debug"
	"github.com/lmorg/ttyphoon/types"
)

const FMT_DATE = "2006.01.02 @ 15.04.05"

//go:embed template_cmd.md
var mdTemplateCmd string

//go:embed template_ai.md
var mdTemplateAi string

type metaT struct {
	AppName      string
	GroupName    string
	TileName     string
	Pwd          string
	Host         string
	TimeStart    string
	TimeEnd      string
	TimeDuration string
	ExitNum      int
	Query        string
	Output       string
}

func Write(tile types.Tile, screen types.Screen) {
	if len(screen) == 0 {
		return
	}

	var err error
	defer func() {
		if err != nil {
			debug.Log(err)
		}
	}()

	var (
		ai   = screen[0].Block.Meta.Is(types.META_BLOCK_AI)
		cmd  string
		data metaT
		tmpl *template.Template
	)

	if ai {
		cmd = "AI Query"
		tmpl, err = template.New("ai").Parse(mdTemplateAi)
		if err != nil {
			return
		}

	} else {
		cmd = firstWord(string(screen[0].Block.Query))
		tmpl, err = template.New("cmd").Parse(mdTemplateCmd)
		if err != nil {
			return
		}
	}

	data = metaT{
		AppName:      app.Name,
		GroupName:    tile.GroupName(),
		TileName:     tile.Name(),
		TimeStart:    screen[0].Block.TimeStart.Format(FMT_DATE),
		TimeEnd:      screen[0].Block.TimeEnd.Format(FMT_DATE),
		TimeDuration: screen[0].Block.TimeEnd.Sub(screen[0].Block.TimeEnd).String(),
		Host:         screen[0].Source.Host,
		Pwd:          screen[0].Source.Pwd,
		Query:        string(screen[0].Block.Query),
		ExitNum:      screen[0].Block.ExitNum,
		Output:       screen.PhraseAll(),
	}

	// auto-hyperlink

	if ai {
		for _, custom := range config.Config.Terminal.Widgets.AutoHyperlink.CustomRegexp {
			if custom.Rx == nil {
				continue
			}

			offset := 0
			posRx := custom.Rx.FindAllStringIndex(data.Output, -1)
			if posRx == nil {
				continue
			}

			var label, link, a, begin, end string

			for i := range posRx {
				label = data.Output[posRx[i][0]+offset : posRx[i][1]+offset]
				link = custom.Rx.ReplaceAllString(label, custom.Link)
				a = fmt.Sprintf(`<a href="%s">%s</a>`, link, label)
				begin = data.Output[:posRx[i][0]+offset]
				end = data.Output[posRx[i][1]+offset:]
				offset += len(a) - len(label)
				data.Output = begin + a + end
			}
		}
	}

	// write

	path := fmt.Sprintf("%s/Documents/%s/history/%s", xdg.Home, app.DirName, data.GroupName)
	err = os.MkdirAll(path, 0700)
	if err != nil {
		return
	}

	filename := fmt.Sprintf("%s/%s-%s.md", path, data.TimeStart, cmd)
	f, err := os.Create(filename)
	if err != nil {
		return
	}
	defer f.Close()

	err = tmpl.Execute(f, data)
}

func firstWord(s string) string {
	if len(s) == 0 {
		return ""
	}

	var i int
	for i = range s {
		if s[i] <= ' ' {
			i--
			break
		}
	}
	return s[:i+1]
}
