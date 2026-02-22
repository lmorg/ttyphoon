package main

import (
	"os"

	"github.com/lmorg/ttyphoon/utils/dispatcher"
)

func main() {
	if build := os.Getenv("MXTTY_BUILD"); build != "" {
		startWails(dispatcher.WindowNameT(build))
		return
	}

	loadEnvs()

	switch dispatcher.WindowNameT(os.Getenv(dispatcher.ENV_WINDOW)) {
	case dispatcher.WindowInputBox:
		startWails(dispatcher.WindowInputBox)

	default:
		startSdl()
	}
}
