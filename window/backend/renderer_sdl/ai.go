package rendersdl

import (
	"fmt"
	"strings"

	"github.com/lmorg/ttyphoon/ai"
	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/ai/skills"
	"github.com/lmorg/ttyphoon/types"
)

func askAi(sr *sdlRender, pos *types.XY) {
	term := sr.termWin.Active.GetTerm()
	meta := agent.Get(sr.termWin.Active.Id())
	//meta.Term = term
	//meta.Renderer = sr
	meta.CmdLine = term.CmdLine(pos)
	meta.Pwd = term.Pwd(pos)
	meta.OutputBlock = ""
	meta.InsertAfterRowId = term.GetRowId(term.GetCursorPosition().Y - 1)

	sr.DisplayInputBox(fmt.Sprintf("What would you like to ask %s?", meta.ServiceName()), "", func(prompt string) {
		ai.AskAI(meta, prompt)
	}, nil)
}

func askAiSkill(sr *sdlRender, pos *types.XY) {
	term := sr.termWin.Active.GetTerm()
	meta := agent.Get(sr.termWin.Active.Id())
	meta.CmdLine = term.CmdLine(pos)
	meta.Pwd = term.Pwd(pos)
	meta.OutputBlock = ""
	meta.InsertAfterRowId = term.GetRowId(term.GetCursorPosition().Y - 1)

	skills := skills.ReadSkills()

	if len(skills) == 0 {
		sr.DisplayNotification(types.NOTIFY_WARN, "You don't have any Agent Skills available to use")
	}

	var padFunc, padName int
	for i := range skills {
		padName = max(padName, len(skills[i].Name))
		padFunc = max(padFunc, len(skills[i].FunctionName))
	}

	slice := make([]string, len(skills))
	for i := range skills {
		slice[i] = fmt.Sprintf("/%s%s%s%s(%s)",
			skills[i].FunctionName,
			strings.Repeat(" ", padFunc+3-len(skills[i].FunctionName)),
			skills[i].Name,
			strings.Repeat(" ", padName+1-len(skills[i].Name)),
			skills[i].Description)
	}

	fnSelect := func(i int) {
		sr.DisplayInputBox(fmt.Sprintf("/%s (%s)", skills[i].FunctionName, skills[i].Description), "", func(prompt string) {
			ai.AskAI(meta, fmt.Sprintf("/%s %s", skills[i].FunctionName, prompt))
		}, nil)
	}

	sr.DisplayMenu("Select an agent skill", slice, nil, fnSelect, nil)
}
