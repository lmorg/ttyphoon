package printing

import (
	"context"
	"errors"
	"testing"
)

func TestPrintRejectsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := Print(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Print() = %v, want context.Canceled", err)
	}
}
