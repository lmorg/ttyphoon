package subagents

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
	"github.com/lmorg/ttyphoon/ai/subagent"
	"github.com/lmorg/ttyphoon/types"
)

const (
	_AGENT_DELEGATE = 1
	_AGENT_REPORT   = 2
)

type Subagent struct {
	agent     aitypes.Agent
	enabled   bool
	agentType int
}

type requestT struct {
	Name   string `json:"name"`
	Prompt string `json:"prompt"`
}

type configT interface {
	ProviderName() string
	ModelName() string
	EnvironmentValue(string) string
}

type delegateToolRunner interface {
	RunSubagentWithTools(context.Context, string, func(string)) (string, error)
}

func init() {
	agent.ToolsAdd(&Subagent{agentType: _AGENT_DELEGATE})
	agent.ToolsAdd(&Subagent{agentType: _AGENT_REPORT})
}

//go:embed delegate_description.md
var delegateDescription string

//go:embed delegate_prompt.md
var delegateSystemPrompt string

//go:embed report_description.md
var reportDescription string

//go:embed report_prompt.md
var reportSystemPrompt string

func (t *Subagent) New(agt aitypes.Agent) (aitypes.Tool, error) {
	return &Subagent{agent: agt, enabled: true, agentType: t.agentType}, nil
}

func (t *Subagent) Enabled() bool { return t.enabled }
func (t *Subagent) Toggle()       { t.enabled = !t.enabled }
func (t *Subagent) Name() string {
	switch t.agentType {
	case _AGENT_DELEGATE:
		return "delegate"
	case _AGENT_REPORT:
		return "report"
	default:
		panic("unknown agent type")
	}
}
func (t *Subagent) Path() string        { return "internal" }
func (t *Subagent) StreamsOutput() bool { return true }
func (t *Subagent) DefaultPermissions() aitypes.DefaultPermissions {
	return aitypes.DefaultPermissions{Invocation: "alwaysAllow", Subagents: "deny"}
}

func (t *Subagent) systemPrompt() string {
	switch t.agentType {
	case _AGENT_DELEGATE:
		return delegateSystemPrompt
	case _AGENT_REPORT:
		return reportSystemPrompt
	default:
		panic("unknown agent type")
	}
}

func (t *Subagent) description() string {
	switch t.agentType {
	case _AGENT_DELEGATE:
		return delegateDescription
	case _AGENT_REPORT:
		return reportDescription
	default:
		panic("unknown agent type")
	}
}

func (t *Subagent) Description() string {
	description := t.description()
	if configured, ok := t.agent.(interface{ SubagentToolNames() []string }); ok {
		if names := configured.SubagentToolNames(); len(names) > 0 {
			return description + "\n\nAllowed sub-agent tools: `" + strings.Join(names, "`, `") + "`."
		}
	}
	return description + "\n\nNo tools are currently allowed for sub-agents."
}

func (t *Subagent) Call(ctx context.Context, input string) (string, error) {
	var requests []*requestT
	if err := json.Unmarshal([]byte(input), &requests); err != nil {
		return "call the tool error: input must be valid json with name and prompt", nil
	}

	var wg sync.WaitGroup
	resp := newResponsesT(len(requests))

	for i, request := range requests {
		wg.Add(1)
		sticky := t.agent.Renderer().DisplaySticky(types.NOTIFY_INFO, "Running subagent: "+request.Name, func() {
			t.agent.Renderer().DisplayNotification(types.NOTIFY_ERROR, fmt.Sprintf("Subagent %s cannot be cancelled", request.Name))
		})
		wg.Go(func() {
			request.Name = strings.TrimSpace(request.Name)
			request.Prompt = strings.TrimSpace(request.Prompt)
			if request.Name == "" || request.Prompt == "" {
				resp.store(i, request.Name, "call the tool error: name and prompt are required", nil)
				return
			}

			configured, ok := t.agent.(configT)
			if !ok {
				resp.store(i, request.Name, "", fmt.Errorf("sub-agent is unavailable for this AI runtime"))
				return
			}
			subagentRequest := subagent.Request{
				Name:         request.Name,
				Prompt:       request.Prompt,
				SystemPrompt: t.systemPrompt(),
				EmitStream:   agent.EmitAIStreamToolProgress(ctx),
			}
			if runner, ok := t.agent.(delegateToolRunner); ok {
				subagentRequest.RunWithTools = runner.RunSubagentWithTools
			}

			s, err := subagent.New(configured.ProviderName(), configured.ModelName(), configured.EnvironmentValue).Run(ctx, subagentRequest)
			resp.store(i, request.Name, s, err)
			sticky.Close()
		})
	}

	wg.Wait()
	return resp.json()
}

type responsesT struct {
	SubAgents []struct {
		Name     string
		Response string
		Error    string
	}
	mu sync.Mutex
}

func newResponsesT(i int) *responsesT {
	var r responsesT
	r.SubAgents = make([]struct {
		Name     string
		Response string
		Error    string
	}, i)
	return &r
}

func (r *responsesT) store(i int, name, s string, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.SubAgents[i].Name = name
	r.SubAgents[i].Response = s
	if err != nil {
		r.SubAgents[i].Error = err.Error()
	}
}

func (r *responsesT) json() (string, error) {
	b, err := json.Marshal(r)
	return string(b), err
}
