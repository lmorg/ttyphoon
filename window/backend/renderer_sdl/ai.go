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
	agt := agent.Get(sr.termWin.Active.Id())
	agt.Meta = &agent.Meta{}

	sr.DisplayInputBox(fmt.Sprintf("What would you like to ask %s?", agt.ServiceName()), "", func(prompt string) {
		ai.AskAI(agt, prompt)
	}, nil)
}

func askAiSkill(sr *sdlRender, pos *types.XY) {
	agt := agent.Get(sr.termWin.Active.Id())
	agt.Meta = &agent.Meta{}

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
			ai.AskAI(agt, fmt.Sprintf("/%s %s", skills[i].FunctionName, prompt))
		}, nil)
	}

	sr.DisplayMenu("Select an agent skill", slice, nil, fnSelect, nil)
}
