package subagent

import (
	"context"
	"strings"
	"testing"

	"github.com/lmorg/ttyphoon/utils/envvars"
)

func TestNewModelRejectsUnsupportedScopedSettings(t *testing.T) {
	env, err := envvars.Resolve(nil, map[string]string{"ANTHROPIC_PROFILE": "profile"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = New(ProviderAnthropic, "model", env).newModel(context.Background())
	if err == nil || !strings.Contains(err.Error(), "ANTHROPIC_PROFILE") {
		t.Fatalf("subagent model error = %v, want scoped profile rejection", err)
	}
}

func TestQuote(t *testing.T) {
	got := Quote("first line\nsecond line\n")
	want := "first line\n> second line\n> "
	if got != want {
		t.Fatalf("Quote() = %q, want %q", got, want)
	}
}

func TestRun_DisableStreamFramingEmitsOnlyContent(t *testing.T) {
	var streamed strings.Builder
	client := &Client{}
	_, err := client.Run(context.Background(), Request{
		Name:                 "worker",
		EmitStream:           func(text string) { streamed.WriteString(text) },
		DisableStreamFraming: true,
		FormatStreamChunk:    func(text string) string { return text },
		RunWithTools: func(_ context.Context, _, _ string, emit func(string)) (string, error) {
			emit("raw output")
			return "raw output", nil
		},
	})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got := streamed.String(); got != "raw output" {
		t.Fatalf("streamed = %q, want raw output", got)
	}
}
