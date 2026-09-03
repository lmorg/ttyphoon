package agent

import (
	"strings"
	"testing"
)

func TestUserQuestionRequestRoundTrip(t *testing.T) {
	requestID, ch := newUserQuestionRequest()
	if requestID == "" || ch == nil {
		t.Fatal("expected request ID and channel")
	}

	markdown := formatUserQuestionRequestMarkdown("Which environment should I use?", requestID, []string{"prod", "staging"})
	if markdown == "" {
		t.Fatal("expected markdown request")
	}
	if !strings.Contains(markdown, "ttyphoon://ai-user-question") {
		t.Fatalf("expected ttyphoon user-question link in markdown, got %q", markdown)
	}

	if err := ResolveUserQuestionRequest(requestID, "staging"); err != nil {
		t.Fatalf("ResolveUserQuestionRequest() returned error: %v", err)
	}

	select {
	case got := <-ch:
		if got != "staging" {
			t.Fatalf("got %q, want %q", got, "staging")
		}
	default:
		t.Fatal("expected decision to be delivered on channel")
	}
}

func TestResolveUserQuestionRequest_NoopWhenRequestAlreadyResolved(t *testing.T) {
	if err := ResolveUserQuestionRequest("missing-request-id", "staging"); err != nil {
		t.Fatalf("ResolveUserQuestionRequest() returned error for a stale request: %v", err)
	}
}
