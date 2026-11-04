package ai

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/lmorg/ttyphoon/ai/agent/sessiondb"
)

func TestAIPromptOutcome(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "success", want: sessiondb.PromptStatusSuccess},
		{name: "failure", err: errors.New("provider failed"), want: sessiondb.PromptStatusFailed},
		{name: "timeout is failure", err: context.DeadlineExceeded, want: sessiondb.PromptStatusFailed},
		{name: "cancelled", err: context.Canceled, want: sessiondb.PromptStatusCancelled},
		{name: "wrapped cancellation", err: fmt.Errorf("provider stopped: %w", context.Canceled), want: sessiondb.PromptStatusCancelled},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := aiPromptOutcome(test.err); got != test.want {
				t.Fatalf("aiPromptOutcome(%v) = %q, want %q", test.err, got, test.want)
			}
		})
	}
}
