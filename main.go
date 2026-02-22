package main

import (
	"os"

	"github.com/lmorg/ttyphoon/utils/dispatcher"
)

func main() {
	if build := os.Getenv("MXTTY_BUILD"); build != "" {
		startWails(dispatcher.WindowTypeT(build))
		return
	}

	loadEnvs()

	switch dispatcher.WindowTypeT(os.Getenv(dispatcher.ENV_WINDOW)) {
	case dispatcher.WindowInputBox:
		startWails(dispatcher.WindowInputBox)

	default:
		startSdl()
	}
}
