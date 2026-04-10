package main

import (
	"strings"

	"github.com/lmorg/murex/utils/which"
	"github.com/lmorg/ttyphoon/tmux"
	"github.com/lmorg/ttyphoon/window/backend"
)

func startBackend(a *WApp) error {
	renderer, size := backend.Initialise()

	if which.Which("tmux") == "" {
		panic("tmux not in $PATH")
	}

	tmuxClient, err := tmux.NewStartSession(renderer, size, tmux.START_ATTACH_SESSION)
	if err != nil {
		if !strings.HasPrefix(err.Error(), "no sessions") {
			//log.Println(err)
			return err
		}

		cdHome()

		tmuxClient, err = tmux.NewStartSession(renderer, size, tmux.START_NEW_SESSION)
		if err != nil {
			//log.Println(err)
			return err
		}
	}

	backend.Start(renderer, tmuxClient.GetTermTiles(), tmuxClient, a.ctx, a)
	return nil
}
