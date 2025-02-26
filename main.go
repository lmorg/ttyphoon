package main

import (
	"log"
	"os/exec"
	"strings"

	"github.com/lmorg/mxtty/config"
	"github.com/lmorg/mxtty/debug/pprof"
	"github.com/lmorg/mxtty/ptty"
	virtualterm "github.com/lmorg/mxtty/term"
	"github.com/lmorg/mxtty/tmux"
	"github.com/lmorg/mxtty/types"
	"github.com/lmorg/mxtty/window/backend"
	"github.com/lmorg/mxtty/window/backend/typeface"
)

func main() {
	pprof.Start()
	defer pprof.CleanUp()

	getFlags()

	typeface.Init()
	err := typeface.Open(
		config.Config.TypeFace.FontName,
		config.Config.TypeFace.FontSize,
	)
	if err != nil {
		panic(err.Error())
	}

	if config.Config.Tmux.Enabled && tmuxInstalled() {
		tmuxSession()
	} else {
		regularSession()
	}
}

func tmuxInstalled() bool {
	path, err := exec.LookPath("tmux")
	installed := path != "" && err == nil
	if !installed {
		// disable tmux if not installed
		config.Config.Tmux.Enabled = false
	}
	return installed
}

func tmuxSession() {
	renderer, size := backend.Initialise()
	defer renderer.Close()

	tmuxClient, err := tmux.NewStartSession(renderer, size, tmux.START_ATTACH_SESSION)
	if err != nil {
		if !strings.HasPrefix(err.Error(), "no sessions") {
			panic(err)
		}

		log.Println("No sessions to attach to. Creating new session.")

		tmuxClient, err = tmux.NewStartSession(renderer, size, tmux.START_NEW_SESSION)
		if err != nil {
			panic(err)
		}
	}

	backend.Start(renderer, tmuxClient.ActiveWindow(), tmuxClient)
}

func regularSession() {
	renderer, size := backend.Initialise()
	defer renderer.Close()

	term := virtualterm.NewTerminal(renderer, size, true)
	pty, err := ptty.NewPty(size)
	if err != nil {
		panic(err)
	}

	win := &types.TermWin{
		Terms:  []types.Term{term},
		Cords:  []*types.XY{{}},
		Active: term,
	}

	term.Start(pty)
	backend.Start(renderer, win, nil)
}
