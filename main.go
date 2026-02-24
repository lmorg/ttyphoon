package main

import (
	"os"

	"github.com/lmorg/ttyphoon/utils/dispatcher"
)

func main() {
	loadEnvs()

	if build := os.Getenv("MXTTY_BUILD"); build != "" {
		startWails(dispatcher.WindowTypeT(build))
		return
	}

	window := dispatcher.WindowTypeT(os.Getenv(dispatcher.ENV_WINDOW))
	if window != "" {
		startWails(window)
	} else {
		startSdl()
	}
}
