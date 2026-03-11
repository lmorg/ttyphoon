package backend

import (
	"github.com/lmorg/ttyphoon/types"
	renderwebkit "github.com/lmorg/ttyphoon/window/backend/renderer_webkit"
)

const EnvRenderer = "MXTTY_RENDERER"

func Initialise() (types.Renderer, *types.XY) {
	//if os.Getenv(EnvRenderer) == "webkit" {
	return renderwebkit.Initialise()
	//}

	//return rendersdl.Initialise()
}

func Start(r types.Renderer, termWin *types.AppWindowTerms, tmuxClient any) {
	r.Start(termWin, tmuxClient)
}
