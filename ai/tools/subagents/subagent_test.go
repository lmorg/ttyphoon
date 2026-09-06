package subagents

import (
	"strings"
	"testing"
)

// Descriptions are hard-wrapped markdown, so match against collapsed whitespace
// rather than letting a line break break the assertion.
func containsPhrase(description, phrase string) bool {
	return strings.Contains(strings.Join(strings.Fields(description), " "), phrase)
}

func TestSubagentToolContracts(t *testing.T) {
	delegate := &Subagent{agentType: _AGENT_DELEGATE}
	if got := delegate.Name(); got != "delegate" {
		t.Fatalf("delegate name = %q, want delegate", got)
	}
	if got := delegate.Description(); !containsPhrase(got, "written summary") {
		t.Fatalf("delegate description = %q, want summary guidance", got)
	}
	if got := delegate.systemPrompt(); got != delegateSystemPrompt {
		t.Fatal("delegate system prompt does not match delegate prompt")
	}

	report := &Subagent{agentType: _AGENT_REPORT}
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
