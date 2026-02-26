package ai

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/ai/prompts"
	"github.com/lmorg/ttyphoon/assets"
	"github.com/lmorg/ttyphoon/types"
)

func Explain(agent *agent.Agent, promptDialogue bool) {
	if !promptDialogue {
		askAI(agent, prompts.GetExplain(agent, ""), fmt.Sprintf("```\n%s\n```", agent.Meta.CmdLine), agent.Meta.CmdLine)
		return
	}

	fn := func(userPrompt string) {
		askAI(agent, prompts.GetExplain(agent, userPrompt), "> "+userPrompt, userPrompt)
	}

	agent.Renderer().DisplayInputBox("(Optional) Add to prompt", "", fn, nil)
}

const _STICKY_MESSAGE = "Asking %s.... "

var _STICKY_SPINNER = []string{
	"🤔", "",
}

func AskAI(agent *agent.Agent, prompt string) {
	go func() {
		askAI(agent, prompts.GetAsk(agent, prompt), "> "+prompt, prompt)
	}()
}

func askAI(agent *agent.Agent, prompt string, title string, query string) {
	insertAfterRowId := agent.Term().GetRowId(agent.Term().GetCursorPosition().Y - 1)
	stickyMessage := fmt.Sprintf(_STICKY_MESSAGE, agent.ServiceName())
	sticky := agent.Renderer().DisplaySticky(types.NOTIFY_INFO, stickyMessage, func() {})
	fin := make(chan struct{})
	var i int

	go func() {
		for {
			select {
			case <-fin:
				sticky.Close()
				return
			case <-time.After(500 * time.Millisecond):
				sticky.SetMessage(fmt.Sprintf("%s %s", stickyMessage, _STICKY_SPINNER[i]))
				agent.Renderer().TriggerRedraw()
				i++
				if i >= len(_STICKY_SPINNER) {
					i = 0
				}
			}
		}
	}()

	go func() {
		result, err := agent.RunLLM(prompt, sticky)
		fin <- struct{}{}
		if err != nil {
			agent.Renderer().DisplayNotification(types.NOTIFY_ERROR, err.Error())
			result = err.Error()

		} else {
			agent.AddHistory(title, result)
		}

		result = fmt.Sprintf("# Your question\n\n%s\n\n# %s's Response\n\n%s", title, agent.ServiceName(), result)

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
			glamour.WithWordWrap(int(agent.Term().GetSize().X)-1),
			glamour.WithStylesFromJSONBytes(theme),
		)
		if err != nil {
			markdown = result
		} else {
			defer md.Close()
			markdown, err = md.Render(result)
			if err != nil {
				markdown = result
			} else {
				// this is a kludge to work around a bug in the markdown package
				markdown = strings.ReplaceAll(markdown, "!```codeblock!start!", "\u001b_begin;code-block\u001b\\")
				markdown = strings.ReplaceAll(markdown, "!```codeblock!end!", "\u001b_end;code-block\u001b\\")
				markdown = strings.ReplaceAll(markdown, "!```mdtable!start!", "\u001b_begin;md-table\u001b\\")
				markdown = strings.ReplaceAll(markdown, "!```mdtable!end!", "\u001b_end;md-table\u001b\\")
			}
		}

		err = agent.Term().InsertSubTerm(query, markdown, insertAfterRowId, types.META_BLOCK_AI)
		if err != nil {
			agent.Renderer().DisplayNotification(types.NOTIFY_ERROR, err.Error())
			return
		}
	}()
}
