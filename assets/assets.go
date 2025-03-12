package assets

/*
	The purpose of this directory is to bundle any external dependency we might
	need, into the terminal's executable.

	This file defines what assets to include and what to ignore.

	All assets have to sit within this directory.
*/

import (
	"bytes"
	"embed"
	"fmt"
)

const (
	BELL = "bell.mp3"

	ICON_APP      = "icon.bmp"
	ICON_DEBUG    = "icon-message.bmp"
	ICON_INFO     = "icon-info.bmp"
	ICON_WARN     = "icon-warn.bmp"
	ICON_ERROR    = "icon-error.bmp"
	ICON_DOWN     = "icon-down.bmp"
	ICON_QUESTION = "icon-question.bmp"

	TYPEFACE    = "Hasklig-Regular.ttf"
	TYPEFACE_B  = "Hasklig-Bold.ttf"
	TYPEFACE_BI = "Hasklig-BoldIt.ttf"
	TYPEFACE_I  = "Hasklig-It.ttf"
	TYPEFACE_L  = "Hasklig-Light.ttf"
	TYPEFACE_LI = "Hasklig-LightIt.ttf"

	EMOJI        = "NotoColorEmoji-emojicompat.ttf"
	FONT_AWESOME = "Font Awesome 6 Free-Solid-900.otf"
)

//go:embed bell.mp3
//go:embed icon.bmp
//go:embed *.ttf
//go:embed *.otf
//go:embed icon-*.bmp
var embedFs embed.FS

var assets map[string][]byte

func init() {
	assets = make(map[string][]byte)

	dir, err := embedFs.ReadDir(".")
	if err != nil {
		panic(err)
	}

	for i := range dir {
		name := dir[i].Name()

		b, err := embedFs.ReadFile(name)
		if err != nil {
			// not a bug in murex
			panic(err)
		}

		assets[name] = b
	}
}

func Get(name string) []byte {
	b, ok := assets[name]
	if !ok {
		panic(fmt.Sprintf("no asset found named '%s'", name))
	}
	return b
}

func Reader(name string) *bytes.Reader {
	return bytes.NewReader(Get(name))
}
