package ai

import (
	"fmt"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/lmorg/mxtty/ai/agent"
	"github.com/lmorg/mxtty/ai/prompts"
	"github.com/lmorg/mxtty/assets"
	"github.com/lmorg/mxtty/types"
)

func Explain(meta *agent.Meta, promptDialogue bool) {
	if !promptDialogue {
		askAI(meta, prompts.GetExplain(meta, ""), fmt.Sprintf("```\n%s\n```", meta.CmdLine), meta.CmdLine)
		return
	}

	fn := func(userPrompt string) {
		askAI(meta, prompts.GetExplain(meta, userPrompt), "> "+userPrompt, userPrompt)
	}

	meta.Renderer.DisplayInputBox("(Optional) Add to prompt", "", fn, nil)
}

const _STICKY_MESSAGE = "Asking %s.... "

var _STICKY_SPINNER = []string{
	"🤔", "",
}

func AskAI(meta *agent.Meta, prompt string) {
	askAI(meta, prompts.GetAsk(meta, prompt), "> "+prompt, prompt)
}

func askAI(meta *agent.Meta, prompt string, title string, query string) {
	stickyMessage := fmt.Sprintf(_STICKY_MESSAGE, meta.ServiceName())
	sticky := meta.Renderer.DisplaySticky(types.NOTIFY_INFO, stickyMessage)
	fin := make(chan struct{})
	var i int

	go func() {
		for {
			select {
			case <-fin:
				sticky.SetMessage("Formatting output....")
				return
			case <-time.After(500 * time.Millisecond):
				sticky.SetMessage(fmt.Sprintf("%s %s", stickyMessage, _STICKY_SPINNER[i]))
				meta.Renderer.TriggerRedraw()
				i++
				if i >= len(_STICKY_SPINNER) {
					i = 0
				}
			}
		}
	}()

	go func() {
		defer sticky.Close()

		result, err := meta.RunLLM(prompt)
		fin <- struct{}{}
		if err != nil {
			meta.Renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
			//return
			result = err.Error()

		} else {
			meta.AddHistory(title, result)
		}

		result = fmt.Sprintf("# Your question:\n\n%s\n\n# %s's Response:\n\n%s", title, meta.ServiceName(), result)

		var (
			markdown string
			theme    []byte
		)
		if types.THEME_LIGHT {
			theme = assets.Get(assets.GLAMOUR_STYLE_LIGHT)
		} else {
			theme = assets.Get(assets.GLAMOUR_STYLE_DARK)
		}

		md, err := glamour.NewTermRenderer(
			glamour.WithEmoji(),
			glamour.WithWordWrap(int(meta.Term.GetSize().X)),
			glamour.WithStylesFromJSONBytes(theme),
		)
		if err != nil {
			markdown = result
		} else {
			defer md.Close()
			markdown, err = md.Render(result)
			if err != nil {
				markdown = result
			}
		}

		err = meta.Term.InsertSubTerm(query, markdown, meta.InsertAfterRowId, types.ROW_OUTPUT_BLOCK_AI)
		if err != nil {
			meta.Renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
			return
		}
	}()
}
