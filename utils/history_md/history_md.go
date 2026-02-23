package historymd

import (
	_ "embed"
	"fmt"
	"os"
	"text/template"

	"github.com/adrg/xdg"
	"github.com/lmorg/ttyphoon/app"
	"github.com/lmorg/ttyphoon/debug"
	"github.com/lmorg/ttyphoon/types"
)

const FMT_DATE = "2006.01.02 @ 15.04.05"

//go:embed template.md
var mdTemplate string

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

	data := metaT{
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

	var cmd string
	if screen[0].Block.Meta.Is(types.META_BLOCK_AI) {
		cmd = "AI Query"
	} else {
		cmd = firstWord(data.Query)
	}

	// write

	path := fmt.Sprintf("%s/Documents/%s/history/%s", xdg.Home, app.DirName, data.GroupName)
	err = os.MkdirAll(path, 0700)
	if err != nil {
		return
	}

	tmpl, err := template.New("md").Parse(mdTemplate)
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
