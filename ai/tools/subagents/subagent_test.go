package subagents

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"

	"github.com/lmorg/ttyphoon/ai/agent"
)

// Descriptions are hard-wrapped markdown, so match against collapsed whitespace
// rather than letting a line break break the assertion.
func containsPhrase(description, phrase string) bool {
	return strings.Contains(strings.Join(strings.Fields(description), " "), phrase)
}

func TestSubagentToolContracts(t *testing.T) {
	delegate := &Subagent{agentType: agent.TOOL_DELEGATE}
	if got := delegate.Name(); got != "delegate" {
		t.Fatalf("delegate name = %q, want delegate", got)
	}
	if got := delegate.Description(); !containsPhrase(got, "written summary") {
		t.Fatalf("delegate description = %q, want summary guidance", got)
	}
	if got := delegate.systemPrompt(); got != delegateSystemPrompt {
		t.Fatal("delegate system prompt does not match delegate prompt")
	}

	report := &Subagent{agentType: agent.TOOL_REPORT}
	if got := report.Name(); got != "report" {
		t.Fatalf("report name = %q, want report", got)
	}
	if got := report.Description(); !containsPhrase(got, "verbatim") {
		t.Fatalf("report description = %q, want verbatim guidance", got)
	}
	if got := report.systemPrompt(); got != reportSystemPrompt {
		t.Fatal("report system prompt does not match report prompt")
	}
}

// Malformed input reaches Call straight from model output, so it must degrade to
// a message rather than panic or abort the run.
func TestCallRejectsMalformedInputWithoutTouchingTheAgent(t *testing.T) {
	tool := &Subagent{agentType: agent.TOOL_DELEGATE} // nil agent: nothing below may dereference it

	for name, input := range map[string]string{
		"not json":     `{"name":"a"}garbage`,
		"json object":  `{"name":"a","prompt":"b"}`,
		"null element": `[null]`,
		"empty name":   `[{"name":"  ","prompt":"b"}]`,
		"empty prompt": `[{"name":"a","prompt":"  "}]`,
		"empty array":  `[]`,
	} {
		t.Run(name, func(t *testing.T) {
			got, err := tool.Call(context.Background(), input)
			if err != nil {
				t.Fatalf("Call returned error %v, want nil so the agent run continues", err)
			}
			if got == "" {
				t.Fatal("Call returned an empty response")
			}
		})
	}
}

func TestCallReportsPerRequestErrors(t *testing.T) {
	tool := &Subagent{agentType: agent.TOOL_DELEGATE}

	got, err := tool.Call(context.Background(), `[{"name":"a","prompt":""},{"name":"","prompt":"b"}]`)
	if err != nil {
		t.Fatalf("Call returned error %v, want nil", err)
	}

	var decoded responsesT
	if err := json.Unmarshal([]byte(got), &decoded); err != nil {
		t.Fatalf("response is not valid json: %v", err)
	}
	if len(decoded.SubAgents) != 2 {
		t.Fatalf("got %d responses, want 2", len(decoded.SubAgents))
	}
	for i, sub := range decoded.SubAgents {
		if !strings.Contains(sub.Response, "name and prompt are required") {
			t.Fatalf("response %d = %q, want a validation message", i, sub.Response)
		}
	}
}

// Results are stored from parallel goroutines and must land on their own index.
func TestResponsesStoreIsIndexedAndConcurrencySafe(t *testing.T) {
	const count = 32
	resp := newResponsesT(count)

	var wg sync.WaitGroup
	for i := range count {
		wg.Go(func() {
			resp.store(i, "agent", "response", nil)
		})
	}
	wg.Wait()

	for i, sub := range resp.SubAgents {
		if sub.Name != "agent" || sub.Response != "response" {
			t.Fatalf("index %d not populated: %+v", i, sub)
		}
	}
}

// The delegation was previously unimplemented: this assertion fails to compile
// if *agent.Agent ever stops satisfying the runner the tool asserts on.
var _ delegateToolRunner = (*agent.Agent)(nil)
