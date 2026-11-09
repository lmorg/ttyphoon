package about

import (
	"bytes"
	"strings"
	"testing"

	"github.com/lmorg/ttyphoon/app"
)

func TestDocumentTemplate(t *testing.T) {
	var rendered bytes.Buffer
	err := documentTemplate.Execute(&rendered, about{
		Version:   "v1.2.3",
		Copyright: "Copyright Example",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"| Version | v1.2.3 |",
		"| Copyright | Copyright Example |",
	} {
		if !strings.Contains(rendered.String(), want) {
			t.Errorf("rendered document does not contain %q", want)
		}
	}
}

func TestMarkdown(t *testing.T) {
	got := Markdown()
	if got != strings.TrimSpace(got) {
		t.Error("Markdown contains leading or trailing whitespace")
	}
	for _, want := range []string{
		"| Version | " + app.Version() + " |",
		"| Copyright | " + app.Copyright() + " |",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("Markdown does not contain %q", want)
		}
	}
}
