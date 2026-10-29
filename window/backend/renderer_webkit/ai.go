package rendererwebkit

import (
	"fmt"
	"strings"

	"github.com/lmorg/ttyphoon/ai"
	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
	"github.com/lmorg/ttyphoon/ai/skills"
	"github.com/lmorg/ttyphoon/types"
)

func (wr *webkitRender) AskAi() {
	//log.Println("[trace] *webkitRender.AskAi()")
	agt := agent.Get(wr.termWin.Active.Id())
	agt.Meta = &aitypes.Meta{}

	wr.DisplayInputBoxW(&types.InputBoxWT{
		Options: types.InputBoxWTOptions{
			Title:           fmt.Sprintf("What would you like to ask %s?", agt.ServiceName()),
			DraftKey:        ai.AIWorkspaceDraftKey(agt),
			Multiline:       true,
			AllowImagePaste: true,
			Variables: []types.InputBoxWTVariables{
				agt.ListModelsInputVariable(),
				ai.EffortInputVariable(),
				ai.SaveMarkdownToggle(false),
			},
		},
		OkFunc: func(v *types.InputBoxCallbackResultT) {
			agt.Meta.Variables = v.Variables
			agt.SetModelFromInputVariable(v.Variables)
			ai.AskAIWithImages(agt, v.String(), v.Images)
		},
	})
}

func askAiSkills(wr *webkitRender) {
	askAiSkillsAt(wr, 0, 0, false)
}

func askAiSkillsAt(wr *webkitRender, x, y int, anchored bool) {
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

	if anchored {
		wr.DisplayMenuAt("Select an agent skill", slice, x, y, nil, fnSelect, nil)
		return
	}
	wr.DisplayMenu("Select an agent skill", slice, nil, fnSelect, nil)
}

func AskAiSkillsAt(x, y int) {
	wr, ok := CurrentRenderer()
	if !ok {
		return
	}
	askAiSkillsAt(wr, x, y, true)
}

func askAiSkill(wr *webkitRender, skill *skills.SkillT) {
	agt := agent.Get(wr.termWin.Active.Id())
	agt.Meta = &aitypes.Meta{}

	parameters := &types.InputBoxWT{
		Options: types.InputBoxWTOptions{
			Title:           strings.Title(skill.Description),
			DraftKey:        ai.AIWorkspaceDraftKey(agt),
			Multiline:       true,
			AllowImagePaste: true,
			Variables: append(append([]types.InputBoxWTVariables(nil), skill.Variables...),
				ai.EffortInputVariable(),
				agt.ListModelsInputVariable(),
				ai.SaveMarkdownToggle(false),
			),
		},
		OkFunc: func(v *types.InputBoxCallbackResultT) {
			agt.Meta.Variables = v.Variables
			agt.SetModelFromInputVariable(v.Variables)
			ai.AskAIWithImages(agt, fmt.Sprintf("/%s %s", skill.FunctionName, v.String()), v.Images)
		},
	}
	wr.DisplayInputBoxW(parameters)
}
