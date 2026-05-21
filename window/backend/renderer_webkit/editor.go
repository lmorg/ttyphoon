package rendererwebkit

import "github.com/lmorg/ttyphoon/types"

func (wr *webkitRender) VisualEditor() {
	parameters := &types.InputBoxWT{
		Options: types.InputBoxWTOptions{
			Title:       "Visual editor",
			Placeholder: "Text to send to terminal",
			Multiline:   true,
		},
		OkFunc: func(value string) {
			if value != "" {
				wr.termWin.Active.GetTerm().Reply([]byte(value))
			}
		},
	}
	wr.DisplayInputBoxW(parameters)
}
