package agent

import (
	"context"
	"strings"
	"testing"
)

func TestToolOutputSummaryPromptIncludesOriginalRequestAndToolContext(t *testing.T) {
	ctx := WithOriginalUserRequest(context.Background(), "Find why the build failed")
	prompt := toolOutputSummaryPrompt(ctx, "readFiles", `{"path":"build.log"}`, "compile error at main.go:42")

	for _, expected := range []string{
		"Original user request:\nFind why the build failed",
		"Tool name: readFiles",
		`Tool input arguments (JSON):` + "\n" + `{"path":"build.log"}`,
		"Tool raw output:\ncompile error at main.go:42",
	} {
		if !strings.Contains(prompt, expected) {
			t.Errorf("summary prompt missing %q: %s", expected, prompt)
		}
	}
}

func TestToolOutputSummaryPromptWithoutRequestOmitsRequestSection(t *testing.T) {
	prompt := toolOutputSummaryPrompt(context.Background(), "readFiles", `{}`, "output")
	if strings.Contains(prompt, "Original user request:") {
		t.Fatalf("summary prompt unexpectedly contains an original request: %s", prompt)
	}
	if !strings.Contains(prompt, "Tool name: readFiles") || !strings.Contains(prompt, "Tool raw output:\noutput") {
		t.Fatalf("summary prompt omitted tool context: %s", prompt)
	}
}
