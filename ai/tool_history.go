package ai

import (
	"context"
	"fmt"
	"strconv"

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
	meta             *AgentMeta
}

func (h ChatHistoryDetail) Description() string {
	return `Returns the the full prompt for a specific chat history index.
Input should be an integer.`
}

func (h ChatHistoryDetail) Name() string {
	return "Chat History"
}

func (h ChatHistoryDetail) Call(ctx context.Context, input string) (string, error) {
	if h.CallbacksHandler != nil {
		h.CallbacksHandler.HandleToolStart(ctx, input)
	}

	var result string

	h.meta.Renderer.DisplayNotification(types.NOTIFY_INFO, fmt.Sprintf("%s is remembering question %s", h.meta.ServiceName(), input))

	i, err := strconv.Atoi(input)
	switch {
	case err != nil:
		result = "ERROR: this tool is expecting an integer."
		goto fin
	case i < 0:
		result = "ERROR: you cannot have negative indexes."
		goto fin
	case i >= len(h.meta.history):
		result = "ERROR: index doesn't match a chat."
		goto fin
	}

	result = fmt.Sprintf(_HISTORY_DETAILED,
		h.meta.history[i].Title,
		h.meta.history[i].OutputBlock,
		h.meta.history[i].Response,
	)

fin:

	if h.CallbacksHandler != nil {
		h.CallbacksHandler.HandleToolEnd(ctx, result)
	}

	return result, nil
}
