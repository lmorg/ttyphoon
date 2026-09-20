package agent

import (
	"context"
	"errors"
	"io"
	"strings"

	"github.com/cloudwego/eino/schema"
	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
)

// RunSubagentWithTools runs a sub-agent prompt through its own react agent,
// restricted to the tools delegable to sub-agents.
//
// The run is deliberately isolated: no session history is loaded, so a sub-agent
// only ever sees its own system prompt and task.
func (agent *Agent) RunSubagentWithTools(ctx context.Context, systemPrompt, prompt string, emit func(string)) (string, error) {
	tools := agent.subagentToolSet()
	if len(tools) == 0 {
		return "", errors.New("no tools are enabled for sub-agents")
	}

	runtime := &einoRuntime{agent: agent, tools: tools}
	if err := runtime.init(); err != nil {
		return "", err
	}

	messages := make([]*schema.Message, 0, 2)
	if strings.TrimSpace(systemPrompt) != "" {
		messages = append(messages, schema.SystemMessage(systemPrompt))
	}
	messages = append(messages, schema.UserMessage(prompt))

	return runtime.runSubagentStream(ctx, messages, emit)
}

func (agent *Agent) subagentToolSet() []aitypes.Tool {
	agent.toolMu.RLock()
	defer agent.toolMu.RUnlock()

	tools := make([]aitypes.Tool, 0, len(agent._tools))
	for _, tool := range agent._tools {
		if agent.toolDelegableToSubagentLocked(tool.Name()) {
			tools = append(tools, tool)
		}
	}
	return tools
}

func (r *einoRuntime) runSubagentStream(ctx context.Context, messages []*schema.Message, emit func(string)) (string, error) {
	if r.agentReact == nil {
		if err := r.init(); err != nil {
			return "", err
		}
	}

	// Routing tool progress through the sub-agent's emitter keeps tool calls
	// inside its own buffered block rather than the parent's stream.
	emitter := &aiStreamEmitter{fn: emit}
	ctx = withAIStreamCallback(ctx, emitter)

	stream, err := r.agentReact.Stream(ctx, messages)
	if err != nil {
		return "", err
	}
	defer stream.Close()

	var response strings.Builder
	for {
		msg, recvErr := stream.Recv()
		if recvErr == io.EOF {
			break
		}
		if recvErr != nil {
			emitter.flush()
			// Hitting the step limit still yields usable work, so report it as
			// content rather than losing the run.
			if isMaxStepError(recvErr) && response.Len() > 0 {
				return response.String(), nil
			}
			return response.String(), recvErr
		}
		if msg == nil || msg.Content == "" {
			continue
		}
		response.WriteString(msg.Content)
		emitter.emitText(msg.Content)
	}
	emitter.flush()

	return response.String(), nil
}
