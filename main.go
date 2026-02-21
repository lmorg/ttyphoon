package main

import (
	"embed"
	"os"

	"github.com/lmorg/ttyphoon/utils/dispatcher"
)

//go:embed all:frontend/dist
var wailsAssets embed.FS

func main() {
	if os.Getenv("MXTTY_BUILD") == "true" {
		wInputBox()
		return
	}

	switch dispatcher.WindowNameT(os.Getenv(dispatcher.ENV_WINDOW)) {
	case dispatcher.WindowInputBox:
		wInputBox()

	case dispatcher.WindowSDL:
		startSdl()

	default:
		dispatcher.Start()
	}
}
