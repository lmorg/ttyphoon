package main

import (
	"os"
	"runtime"

	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/utils/dispatcher"
)

func main() {
	loadEnvs()

	window := dispatcher.WindowTypeT(os.Getenv(dispatcher.ENV_WINDOW))
	if window != "" {
		startTerm()
		startWails(window)
	} else {
		startSdl()
	}
}

func startTerm() {
	if runtime.GOOS == "darwin" {
		err := os.Setenv("PATH", "PATH="+os.Getenv("PATH")+":/usr/bin:/opt/homebrew/bin:/opt/homebrew/sbin")
		if err != nil {
			panic(err)
		}
	}

	if config.Config.Tmux.Enabled && tmuxInstalled() {
		tmuxSession()
	} else {
		regularSession()
	}
}
