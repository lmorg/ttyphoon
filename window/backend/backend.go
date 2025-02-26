package backend

import (
	"github.com/lmorg/mxtty/types"
	rendersdl "github.com/lmorg/mxtty/window/backend/renderer_sdl"
)

func Initialise() (types.Renderer, *types.XY) {
	return rendersdl.Initialise()
}

func Start(r types.Renderer, term *types.TermWin, tmuxClient any) {
	r.Start(term, tmuxClient)
}
