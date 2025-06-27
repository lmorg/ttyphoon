package tools

import (
	"context"
	"fmt"
	"strconv"

	"github.com/lmorg/mxtty/ai/agent"
	"github.com/lmorg/mxtty/types"
	"github.com/tmc/langchaingo/callbacks"
)

const _HISTORY_DETAILED = `
## Query summary:
%s
---
## Output block:
%s
---
## Your response:
%s
`

type ChatHistoryDetail struct {
	CallbacksHandler callbacks.Handler
	meta             *agent.Meta
	enabled          bool
}

func init() {
	agent.ToolsAdd(&ChatHistoryDetail{})
}

func (t *ChatHistoryDetail) New(meta *agent.Meta) (agent.Tool, error) {
	return &ChatHistoryDetail{meta: meta, enabled: true}, nil
}

func (t *ChatHistoryDetail) Enabled() bool { return t.enabled }
func (t *ChatHistoryDetail) Toggle()       { t.enabled = !t.enabled }

func (t *ChatHistoryDetail) Description() string {
	return `Returns the the full prompt for a specific chat history index.
Input should be an integer.`
}

func (t *ChatHistoryDetail) Name() string { return "Chat History" }
func (t *ChatHistoryDetail) Path() string { return "internal" }

func (t *ChatHistoryDetail) Call(ctx context.Context, input string) (string, error) {
	if t.CallbacksHandler != nil {
		t.CallbacksHandler.HandleToolStart(ctx, input)
	}

	var result string

	t.meta.Renderer.DisplayNotification(types.NOTIFY_INFO, fmt.Sprintf("%s is remembering question %s", t.meta.ServiceName(), input))

	i, err := strconv.Atoi(input)
	switch {
	case err != nil:
		result = "ERROR: this tool is expecting an integer."
		goto fin
	case i < 0:
		result = "ERROR: you cannot have negative indexes."
		goto fin
	case i >= len(t.meta.History):
		result = "ERROR: index doesn't match a chat."
		goto fin
	}

	result = fmt.Sprintf(_HISTORY_DETAILED,
		t.meta.History[i].Title,
		t.meta.History[i].OutputBlock,
		t.meta.History[i].Response,
	)

fin:

	if t.CallbacksHandler != nil {
		t.CallbacksHandler.HandleToolEnd(ctx, result)
	}

	return result, nil
}
