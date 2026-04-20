package rendererwebkit

import (
	"fmt"
	"strings"

	"github.com/lmorg/ttyphoon/ai"
	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/ai/skills"
	"github.com/lmorg/ttyphoon/types"
)

func askAi(wr *webkitRender) {
	agt := agent.Get(wr.termWin.Active.Id())
	agt.Meta = &agent.Meta{}

	wr.DisplayInputBoxW(&DisplayInputBoxWT{
		Options: DisplayInputBoxWTOptions{
			Title:     fmt.Sprintf("What would you like to ask %s?", agt.ServiceName()),
			Multiline: true, // Enable multiline input for AI prompt
		},
		OkFunc: func(prompt string) {
			ai.AskAI(agt, prompt)
		},
	})
}

func askAiSkills(wr *webkitRender) {
	skills := skills.ReadSkills()

	if len(skills) == 0 {
		wr.DisplayNotification(types.NOTIFY_WARN, "You don't have any Agent Skills available to use")
		return
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
		askAiSkill(wr, skills[i])
	}

	wr.DisplayMenu("Select an agent skill", slice, nil, fnSelect, nil)
}

func askAiSkill(wr *webkitRender, skill *skills.SkillT) {
	agt := agent.Get(wr.termWin.Active.Id())
	agt.Meta = &agent.Meta{}

	parameters := &DisplayInputBoxWT{
		Options: DisplayInputBoxWTOptions{
			Title:     strings.Title(skill.Description),
			Multiline: true,
		},
		OkFunc: func(prompt string) {
			ai.AskAI(agt, fmt.Sprintf("/%s %s", skill.FunctionName, prompt))
		},
	}
	wr.DisplayInputBoxW(parameters)
}
