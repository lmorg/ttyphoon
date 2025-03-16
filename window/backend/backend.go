package backend

import (
	"github.com/lmorg/mxtty/types"
	rendersdl "github.com/lmorg/mxtty/window/backend/renderer_sdl"
)

func Initialise() (types.Renderer, *types.XY) {
	return rendersdl.Initialise()
}

func Start(r types.Renderer, termWin *types.AppWindowTerms, tmuxClient any) {
	r.Start(termWin, tmuxClient)
}
