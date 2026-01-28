package rendersdl

import (
	"fmt"

	"github.com/lmorg/ttyphoon/ai"
	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/types"
)

func askAi(sr *sdlRender, pos *types.XY) {
	term := sr.termWin.Active.GetTerm()
	meta := agent.Get(sr.termWin.Active.Id())
	meta.Term = term
	meta.Renderer = sr
	meta.CmdLine = term.CmdLine(pos)
	meta.Pwd = term.Pwd(pos)
	meta.OutputBlock = ""
	meta.InsertAfterRowId = term.GetRowId(term.GetCursorPosition().Y - 1)

	sr.DisplayInputBox(fmt.Sprintf("What would you like to ask %s?", meta.ServiceName()), "", func(prompt string) {
		ai.AskAI(meta, prompt)
	}, nil)
}
