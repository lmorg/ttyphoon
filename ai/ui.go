package ai

import (
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/lmorg/mxtty/types"
	"github.com/tmc/langchaingo/llms"
)

const (
	LLM_OPENAI    = "ChatGPT"
	LLM_ANTHROPIC = "Claude"
)

var UseService = LLM_ANTHROPIC

type Meta struct {
	Term         types.Term
	Renderer     types.Renderer
	CmdLine      string
	Pwd          string
	OutputBlock  string
	InsertRowPos int32
}

func Explain(meta *Meta, promptDialogue bool) {
	if !promptDialogue {
		explain(meta, "")
		return
	}

	fn := func(userPrompt string) {
		explain(meta, userPrompt)
	}

	meta.Renderer.DisplayInputBox("Custom prompt", "", fn)
}

const _STICKY_MESSAGE = "Generating AI-powered explanation.... (this can take up to a minute)"

var _STICKY_SPINNER = []string{
	"🤔", "",
}

func explain(meta *Meta, userPrompt string) {
	sticky := meta.Renderer.DisplaySticky(types.NOTIFY_INFO, _STICKY_MESSAGE)
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

		var (
			model llms.Model
			err   error
		)

		switch UseService {
		case LLM_ANTHROPIC:
			model, err = llmAnthropic()
		case LLM_OPENAI:
			model, err = llmOpenAI()
		default:
			panic("unexpected branch")
		}
		if err != nil {
			meta.Renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
			return
		}

		result, err := RunLLM(model, meta, userPrompt)
		fin <- struct{}{}
		if err != nil {
			meta.Renderer.DisplayNotification(types.NOTIFY_ERROR, err.Error())
			return
		}

		if userPrompt == "" {
			result = fmt.Sprintf("# %s's Explanation:\n```\n%s\n```\n%s", UseService, meta.CmdLine, result)
		} else {
			result = fmt.Sprintf("# %s's Explanation:\n```\n%s\n```\n# %s\n%s", UseService, meta.CmdLine, userPrompt, result)
		}

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
