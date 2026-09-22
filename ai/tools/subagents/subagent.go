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
	"github.com/lmorg/ttyphoon/ai/agent/sessiondb"
	"github.com/lmorg/ttyphoon/ai/subagent"
	"github.com/lmorg/ttyphoon/types"
)

type Subagent struct {
	agent     aitypes.Agent
	enabled   bool
	agentType string
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
	RunSubagentWithTools(ctx context.Context, systemPrompt, prompt string, emit func(string)) (string, error)
}

func init() {
	agent.ToolsAdd(&Subagent{agentType: agent.TOOL_DELEGATE})
	agent.ToolsAdd(&Subagent{agentType: agent.TOOL_REPORT})
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
func (t *Subagent) Name() string  { return t.agentType }

func (t *Subagent) Path() string        { return "internal" }
func (t *Subagent) StreamsOutput() bool { return true }
func (t *Subagent) DefaultPermissions() aitypes.DefaultPermissions {
	return aitypes.DefaultPermissions{Invocation: "alwaysAllow", Subagents: "deny"}
}

func (t *Subagent) systemPrompt() string {
	switch t.agentType {
	case agent.TOOL_DELEGATE:
		return delegateSystemPrompt
	case agent.TOOL_REPORT:
		return reportSystemPrompt
	default:
		panic("unknown agent type")
	}
}

func (t *Subagent) description() string {
	switch t.agentType {
	case agent.TOOL_DELEGATE:
		return delegateDescription
	case agent.TOOL_REPORT:
		return reportDescription
	default:
		panic("unknown agent type")
	}
}

func (t *Subagent) subagentToolNames() []string {
	if configured, ok := t.agent.(interface{ SubagentToolNames() []string }); ok {
		return configured.SubagentToolNames()
	}
	return nil
}

func (t *Subagent) Description() string {
	description := t.description()
	if names := t.subagentToolNames(); len(names) > 0 {
		return description + "\n\nAllowed sub-agent tools: `" + strings.Join(names, "`, `") + "`."
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
	emitToPanel := agent.EmitAIStreamToolProgress(ctx)

	for i, request := range requests {
		// A JSON null element decodes to a nil pointer.
		if request == nil {
			resp.store(i, "", "call the tool error: name and prompt are required", nil)
			continue
		}

		request.Name = strings.TrimSpace(request.Name)
		request.Prompt = strings.TrimSpace(request.Prompt)

		if request.Name == "" || request.Prompt == "" {
			resp.store(i, request.Name, "call the tool error: name and prompt are required", nil)
			continue
		}

		sticky := t.agent.Renderer().DisplaySticky(types.NOTIFY_INFO, "Running subagent: "+request.Name, func() {})
		wg.Go(func() {
			defer sticky.Close()

			configured, ok := t.agent.(configT)
			if !ok {
				resp.store(i, request.Name, "", fmt.Errorf("sub-agent is unavailable for this AI runtime"))
				return
			}

			// Sub-agents run in parallel but the panel is one linear stream, so
			// buffer each job and emit it as a single contiguous block.
			var (
				block   strings.Builder
				blockMu sync.Mutex
			)
			subagentRequest := subagent.Request{
				Name:         request.Name,
				Prompt:       request.Prompt,
				SystemPrompt: t.systemPrompt(),
				StreamPrefix: "",
				StreamSuffix: "",
				FormatStreamChunk: func(text string) string {
					return text
				},
				EmitStream: func(chunk string) {
					blockMu.Lock()
					block.WriteString(chunk)
					blockMu.Unlock()
				},
			}
			// With no delegable tools, fall through to the toolless sub-agent
			// rather than failing the request.
			if runner, ok := t.agent.(delegateToolRunner); ok && len(t.subagentToolNames()) > 0 {
				subagentRequest.RunWithTools = runner.RunSubagentWithTools
			}

			s, err := subagent.New(configured.ProviderName(), configured.ModelName(), configured.EnvironmentValue).Run(ctx, subagentRequest)
			resp.store(i, request.Name, s, err)

			blockMu.Lock()
			buffered := block.String()
			blockMu.Unlock()
			if buffered != "" && emitToPanel != nil {
				legacy := fmt.Sprintf("\n> **Sub-agent %s:** %s\n\n", request.Name, strings.ReplaceAll(buffered, "\n", "\n> "))
				agent.EmitAIStreamBlockWithLegacy(ctx, sessiondb.StreamBlockSubagent, buffered, legacy)
			}
		})
	}

	wg.Wait()

	s, err := resp.json()
	if err != nil {
		// A tool error must not abort the agent run.
		return fmt.Sprintf("call the tool error: cannot encode sub-agent responses: %s", err), nil
	}
	return s, nil
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
