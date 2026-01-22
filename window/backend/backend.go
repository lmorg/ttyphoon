package backend

import (
	"github.com/lmorg/ttyphoon/types"
	rendersdl "github.com/lmorg/ttyphoon/window/backend/renderer_sdl"
)

func Initialise() (types.Renderer, *types.XY) {
	return rendersdl.Initialise()
}

func Start(r types.Renderer, termWin *types.AppWindowTerms, tmuxClient any) {
	r.Start(termWin, tmuxClient)
}
