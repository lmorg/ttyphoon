package about

import (
	"bytes"
	_ "embed"
	"log"
	"strings"
	"text/template"

	"github.com/lmorg/ttyphoon/app"
)

//go:embed about.md
var document string

type about struct {
	Name    string
	TagLine string

	Version   string
	Copyright string
	License   string

	Branch    string
	BuildDate string
}

var documentTemplate = template.Must(template.New("about.md").Parse(document))

func Markdown() string {
	data := about{
		Name:    app.Name(),
		TagLine: app.TagLine(),

		Version:   app.Version(),
		Copyright: app.Copyright(),
		License:   app.License(),

		Branch:    app.Branch(),
		BuildDate: app.BuildDate(),
	}

	var rendered bytes.Buffer
	if err := documentTemplate.Execute(&rendered, data); err != nil {
		log.Printf("[error] %v", err)
	}
	return strings.TrimSpace(rendered.String())
}
