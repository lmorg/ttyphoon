package main

import (
	"os"

	"github.com/lmorg/ttyphoon/utils/dispatcher"
)

func main() {
	loadEnvs()

	window := dispatcher.WindowTypeT(os.Getenv(dispatcher.ENV_WINDOW))
	if window != "" {
		startWails(window)
	} else {
		startSdl()
	}
}
