package rendererwebkit

func (wr *webkitRender) VisualEditor() {
	parameters := &DisplayInputBoxWT{
		Options: DisplayInputBoxWTOptions{
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
