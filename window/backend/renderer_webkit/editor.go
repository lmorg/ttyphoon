package rendererwebkit

import "github.com/lmorg/ttyphoon/types"

func (wr *webkitRender) VisualEditor() {
	parameters := &types.InputBoxWT{
		Options: types.InputBoxWTOptions{
			Title:       "Visual editor",
			Placeholder: "Text to send to terminal",
			Multiline:   true,
		},
		OkFunc: func(v *types.InputBoxCallbackResultT) {
			if v.Value != "" {
				wr.termWin.Active.GetTerm().Reply([]byte(v.Value))
			}
		},
	}
	wr.DisplayInputBoxW(parameters)
}
