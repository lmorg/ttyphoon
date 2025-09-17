package markdown_test

import (
	"testing"

	_ "embed"

	"github.com/lmorg/mxtty/utils/markdown"
)

//go:embed testdata/sample.md
var sampleMD string

func TestToMarkdown(t *testing.T) {
	md := markdown.Parse(sampleMD)
	s := markdown.ToMarkdown(md)

	if sampleMD != s {
		t.Error("ToMarkdown doesn't match markdown input")
		t.Log(s)
	}
}
