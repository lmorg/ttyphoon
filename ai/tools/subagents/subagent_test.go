package subagents

import (
	"strings"
	"testing"
)

func TestSubagentToolContracts(t *testing.T) {
	delegate := &Subagent{agentType: _AGENT_DELEGATE}
	if got := delegate.Name(); got != "delegate" {
		t.Fatalf("delegate name = %q, want delegate", got)
	}
	if got := delegate.Description(); !strings.Contains(got, "written summary") {
		t.Fatalf("delegate description = %q, want summary guidance", got)
	}
	if got := delegate.systemPrompt(); got != delegateSystemPrompt {
		t.Fatal("delegate system prompt does not match delegate prompt")
	}

	report := &Subagent{agentType: _AGENT_REPORT}
	if got := report.Name(); got != "report" {
		t.Fatalf("report name = %q, want report", got)
	}
	if got := report.Description(); !strings.Contains(got, "verbatim") {
		t.Fatalf("report description = %q, want verbatim guidance", got)
	}
	if got := report.systemPrompt(); got != reportSystemPrompt {
		t.Fatal("report system prompt does not match report prompt")
	}
}
