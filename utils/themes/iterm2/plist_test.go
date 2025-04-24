package iterm2

import (
	"bytes"
	_ "embed"
	"testing"

	"github.com/lmorg/murex/utils/json"
)

//go:embed test_files/Afterglow.itermcolors
var AfterglowItermcolors []byte

//go:embed test_files/Afterglow.expected.json
var AfterglowExpected string

//go:embed test_files/CGA.itermcolors
var CgaItermcolors []byte

//go:embed test_files/CGA.expected.json
var CgaExpected string

func TestUnmarshalThemeAfterglow(t *testing.T) {
	reader := bytes.NewReader(AfterglowItermcolors)
	theme, err := unmarshalTheme(reader)

	if err != nil {
		t.Error(err)
		t.Fail()
	}

	if json.LazyLoggingPretty(theme) != AfterglowExpected {
		t.Error("Parser failed. Got:")
		t.Error(json.LazyLoggingPretty(theme))
	}
}

func TestUnmarshalThemeCga(t *testing.T) {
	reader := bytes.NewReader(CgaItermcolors)
	theme, err := unmarshalTheme(reader)

	if err != nil {
		t.Error(err)
		t.Fail()
	}

	if json.LazyLoggingPretty(theme) != CgaExpected {
		t.Error("Parser failed. Got:")
		t.Error(json.LazyLoggingPretty(theme))
	}
}
