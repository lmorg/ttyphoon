package ai

import (
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/lmorg/mxtty/types"
)

func Explain(term types.Term, renderer types.Renderer, cmdLine, outputBlock string, insertAtCellY int32, promptDialogue bool) {
	if !promptDialogue {
		explain(term, renderer, cmdLine, outputBlock, insertAtCellY, "")
		return
	}

	fn := func(userPrompt string) {
		explain(term, renderer, cmdLine, outputBlock, insertAtCellY, userPrompt)
	}

	renderer.DisplayInputBox("Custom prompt", "", fn)
}

const _STICKY_MESSAGE = "Generating AI-powered explanation.... (this can take up to a minute)"

var _STICKY_SPINNER = []string{
	"🤔", "",
}

func explain(term types.Term, renderer types.Renderer, cmdLine, outputBlock string, insertAtCellY int32, userPrompt string) {
	sticky := renderer.DisplaySticky(types.NOTIFY_INFO, _STICKY_MESSAGE)
	fin := make(chan struct{})
	var i int

	go func() {
		for {
			select {
			case <-fin:
				sticky.SetMessage("Formatting output....")
				return
			case <-time.After(500 * time.Millisecond):
				sticky.SetMessage(fmt.Sprintf("%s %s", _STICKY_MESSAGE, _STICKY_SPINNER[i]))
				renderer.TriggerRedraw()
				i++
				if i >= len(_STICKY_SPINNER) {
					i = 0
				}
			}
		}
	}()

	go func() {
		defer sticky.Close()

		result, err := OpenAI(cmdLine, outputBlock, userPrompt)
		fin <- struct{}{}
		if err != nil {
			renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
			return
		}

		if userPrompt == "" {
			result = fmt.Sprintf("# AI Explanation:\n```\n%s\n```\n%s", cmdLine, result)
		} else {
			result = fmt.Sprintf("# AI Explanation:\n```\n%s\n```\n# %s\n%s", cmdLine, userPrompt, result)
		}

		theme := "dark"
		if types.THEME_LIGHT {
			theme = "light"
		}

		markdown, err := glamour.Render(result, theme)
		if err != nil {
			markdown = result
		}

		//markdown = strings.TrimSpace(markdown)
		//markdown = strings.ReplaceAll(markdown, "\n  ", "\n")

		err = term.InsertSubTerm(markdown, insertAtCellY, types.ROW_OUTPUT_BLOCK_AI)
		if err != nil {
			renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
			return
		}
	}()
}

const _OPENAI_ENV_VAR = "OPENAI_API_KEY"

func EnvOpenAi(renderer types.Renderer) {
	renderer.DisplayInputBox("OpenAI API Key", "", func(s string) {
		_ = os.Setenv(_OPENAI_ENV_VAR, s)
	})
}
