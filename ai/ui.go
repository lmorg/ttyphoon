package ai

import (
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/lmorg/mxtty/types"
)

func (meta *AgentMeta) Explain(promptDialogue bool) {
	if !promptDialogue {
		askAI(meta, meta.explainPrompt(meta.CmdLine, meta.OutputBlock, ""), fmt.Sprintf("```\n%s\n```", meta.CmdLine))
		return
	}

	fn := func(userPrompt string) {
		askAI(meta, meta.explainPrompt(meta.CmdLine, meta.OutputBlock, userPrompt), "> "+userPrompt)
	}

	meta.Renderer.DisplayInputBox("Add to prompt", "", fn)
}

const _STICKY_MESSAGE = "Asking %s.... "

var _STICKY_SPINNER = []string{
	"🤔", "",
}

func (meta *AgentMeta) AskAI(prompt string) {
	askAI(meta, meta.askPrompt(prompt), "> "+prompt)
}

func askAI(meta *AgentMeta, prompt string, title string) {
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

		result, err := meta.runLLM(prompt)
		fin <- struct{}{}
		if err != nil {
			meta.Renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
			//return
			result = err.Error()
		}

		meta.AddHistory(title, result)

		result = fmt.Sprintf("# Your question:\n\n%s\n\n# %s's Explanation:\n\n%s", title, meta.ServiceName(), result)

		theme := "dark"
		if types.THEME_LIGHT {
			theme = "light"
		}

		markdown, err := glamour.Render(result, theme)
		if err != nil {
			markdown = result
		}

		err = meta.Term.InsertSubTerm(markdown, meta.InsertRowPos, types.ROW_OUTPUT_BLOCK_AI)
		if err != nil {
			meta.Renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
			return
		}
	}()
}

const (
	_ANTHROPIC_ENV_VAR = "ANTHROPIC_API_KEY"
	_OPENAI_ENV_VAR    = "OPENAI_API_KEY"
)

func EnvOpenAi(renderer types.Renderer) {
	renderer.DisplayInputBox("OpenAI (ChatGPT) API Key", "", func(s string) {
		_ = os.Setenv(_OPENAI_ENV_VAR, s)
	})
}

func EnvAnthropic(renderer types.Renderer) {
	renderer.DisplayInputBox("Anthropic (Claude) API Key", "", func(s string) {
		_ = os.Setenv(_ANTHROPIC_ENV_VAR, s)
	})
}
