package historymd

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"strings"
	"text/template"

	"github.com/adrg/xdg"
	"github.com/lmorg/ttyphoon/app"
	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/types"
)

const FMT_DATE = "2006.01.02 @ 15.04.05"

//go:embed template_cmd.md
var mdTemplateCmd string

//go:embed template_ai.md
var mdTemplateAi string

type TemplateFieldsT struct {
	filename     string
	AppName      string
	GroupName    string
	TileName     string
	Pwd          string
	Host         string
	TimeStart    string
	TimeEnd      string
	TimeDuration string
	ExitNum      int
	Agent        string
	Query        string
	FullPrompt   string
	Output       string
	OutputLang   string
}

type TemplateWriterT func(tmpl *template.Template, data *TemplateFieldsT) error

func Block(tile types.Tile, screen types.Screen, write TemplateWriterT) error {
	if len(screen) == 0 {
		return errors.New("invalid block")
	}
	if screen[0].Block.Meta.Is(types.META_BLOCK_AI) {
		return blockAi(tile, screen, write)
	}

	cmd := firstWord(string(screen[0].Block.Query))
	tmpl, err := template.New("cmd").Parse(mdTemplateCmd)
	if err != nil {
		return err
	}

	cmdLine := string(screen[0].Block.Query)

	data := &TemplateFieldsT{
		filename:     cmd,
		AppName:      app.Name,
		GroupName:    tile.GroupName(),
		TileName:     tile.Name(),
		TimeStart:    screen[0].Block.TimeStart.Format(FMT_DATE),
		TimeEnd:      screen[0].Block.TimeEnd.Format(FMT_DATE),
		TimeDuration: screen[0].Block.TimeEnd.Sub(screen[0].Block.TimeStart).String(),
		Host:         screen[0].Source.Host,
		Pwd:          screen[0].Source.Pwd,
		Query:        cmdLine,
		ExitNum:      screen[0].Block.ExitNum,
		Output:       screen.PhraseAll(),
	}

	switch {
	case strings.HasPrefix(cmdLine, "diff"):
		data.OutputLang = "diff"
	case strings.HasPrefix(cmdLine, "git diff"):
		data.OutputLang = "diff"
	default:
	}

	return write(tmpl, data)
}

func blockAi(tile types.Tile, screen types.Screen, write TemplateWriterT) error {
	cmd := "AI Query"

	if screen[0].Block.AiMeta == nil {
		return errors.New("nil pointer for AiMeta struct")
	}

	data := &TemplateFieldsT{
		filename:     cmd,
		AppName:      app.Name,
		GroupName:    tile.GroupName(),
		TileName:     tile.Name(),
		TimeStart:    screen[0].Block.TimeStart.Format(FMT_DATE),
		TimeEnd:      screen[0].Block.TimeEnd.Format(FMT_DATE),
		TimeDuration: screen[0].Block.TimeEnd.Sub(screen[0].Block.TimeStart).String(),
		Pwd:          screen[0].Source.Pwd,
		Agent:        screen[0].Block.AiMeta.Agent,
		Query:        string(screen[0].Block.Query),
		FullPrompt:   *screen[0].Block.AiMeta.Prompt,
		Output:       *screen[0].Block.AiMeta.Response,
	}

	// auto-hyperlink

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

	return Ai(data, write)
}

func Ai(data *TemplateFieldsT, write TemplateWriterT) error {
	tmpl, err := template.New("ai").Parse(mdTemplateAi)
	if err != nil {
		return err
	}

	return write(tmpl, data)
}

func TemplateWriter(tmpl *template.Template, data *TemplateFieldsT) error {
	path := fmt.Sprintf("%s/Documents/%s/history/%s", xdg.Home, app.DirName, data.GroupName)
	err := os.MkdirAll(path, 0700)
	if err != nil {
		return err
	}

	filename := fmt.Sprintf("%s/%s-%s.md", path, data.TimeStart, data.filename)
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	return tmpl.Execute(f, data)
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
