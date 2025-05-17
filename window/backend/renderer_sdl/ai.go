package rendersdl

import (
	"fmt"

	"github.com/lmorg/mxtty/ai"
	"github.com/lmorg/mxtty/types"
)

func askAi(sr *sdlRender, pos *types.XY) {
	term := sr.termWin.Active.GetTerm()
	meta := &ai.Meta{
		Term:         term,
		Renderer:     sr,
		CmdLine:      term.CmdLine(pos),
		Pwd:          term.Pwd(pos),
		OutputBlock:  "",
		InsertRowPos: term.ConvertRelativeToAbsoluteY(term.GetSize()) - 1,
	}
	sr.DisplayInputBox(fmt.Sprintf("What would you like to ask %s?", ai.Service()), "", func(prompt string) {
		ai.AskAI(meta, prompt)
	})
}
