package main

import (
	"bufio"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/debug/pprof"
	"github.com/lmorg/ttyphoon/tmux"
	"github.com/lmorg/ttyphoon/utils/file"
	"github.com/lmorg/ttyphoon/window/backend"
	"github.com/lmorg/ttyphoon/window/backend/typeface"
)

func main() {
	pprof.Start()
	defer pprof.CleanUp()

	loadEnvs()

	getFlags()

	typeface.Init()

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

	backend.Start(renderer, tmuxClient.GetTermTiles(), tmuxClient)
}

func regularSession() {
	/*
	   renderer, size := backend.Initialise()
	   defer renderer.Close()

	   	tile := &types.Tile{
	   		Right:  size.X,
	   		Bottom: size.Y,
	   	}

	   virtualterm.NewTerminal(tile, renderer, size, true)
	   pty, err := ptty.NewPty(size)

	   	if err != nil {
	   		panic(err)
	   	}

	   	appWin := &types.AppWindowTerms{
	   		Tiles:  []*types.Tile{tile},
	   		Active: tile,
	   	}

	   tile.Term.Start(pty)
	   backend.Start(renderer, appWin, nil)
	*/
}

func loadEnvs() {
	files := file.GetConfigFiles("/", ".env")
	for i := range files {
		f, err := os.Open(files[i])
		if err != nil {
			log.Print(err)
			continue
		}
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			split := strings.SplitN(scanner.Text(), "=", 2)
			if len(split) != 2 {
				split = []string{files[i], ""}
			}
			log.Printf(`%s: "%s" = "%s"`, files[i], split[0], split[1])
			os.Setenv(split[0], split[1])
		}
	}
}
