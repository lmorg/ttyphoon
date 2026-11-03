package shell

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
	"github.com/lmorg/ttyphoon/utils/markdownutil"
)

type Bash struct {
	agent   aitypes.Agent
	enabled bool
}

func init() {
	agent.ToolsAdd(&Bash{})
}

//go:embed bash_description.md
var bashDescription string

func (t *Bash) New(agt aitypes.Agent) (aitypes.Tool, error) {
	return &Bash{agent: agt, enabled: true}, nil
}

func (t *Bash) Enabled() bool { return t.enabled }
func (t *Bash) Toggle()       { t.enabled = !t.enabled }
func (t *Bash) Name() string  { return "bash" }
func (t *Bash) Path() string  { return "internal" }
func (t *Bash) Description() string {
	return bashDescription
}
func (t *Bash) DefaultPermissions() aitypes.DefaultPermissions {
	return aitypes.DefaultPermissions{Invocation: "askPermission", Subagents: "deny"}
}

type bashInputT struct {
	Script  string `json:"script"`
	Summary string `json:"summary"`
}

type bashResultT struct {
	Output   string `json:"output"`
	ExitCode *int   `json:"exitCode,omitempty"`
	Error    string `json:"error,omitempty"`
}

type bashPaneCloner interface {
	ClonePane(string, func([]byte, int, error)) error
}

func (t *Bash) InputType() reflect.Type { return reflect.TypeOf(bashInputT{}) }

func (t *Bash) CallStructured(ctx context.Context, input any) (string, error) {
	request, ok := input.(*bashInputT)
	if !ok || request == nil {
		return "", fmt.Errorf("bash received an invalid structured input %T", input)
	}
	result, err := t.run(ctx, request.Script)
	if err != nil {
		result.Error = err.Error()
	}
	output, err := json.Marshal(result)
	return string(output), err
}

func (t *Bash) CallDescription(ctx context.Context, input any) string {
	request, ok := input.(*bashInputT)
	if !ok || request == nil {
		return "Tool received an invalid structured input"
	}
	description := markdownutil.MarkdownCodeBlock(request.Script, "bash")
	summary := strings.Join(strings.Fields(request.Summary), " ")
	if summary != "" {
		description = markdownutil.EscapeMarkdownText(summary) + "\n\n" + description
	}
	return description
}

func (t *Bash) run(ctx context.Context, script string) (bashResultT, error) {
	if err := ctx.Err(); err != nil {
		return bashResultT{}, err
	}
	if t.agent == nil {
		return bashResultT{}, fmt.Errorf("terminal pane cloning is unavailable")
	}
	cloner, ok := t.agent.Renderer().(bashPaneCloner)
	if !ok {
		return bashResultT{}, fmt.Errorf("terminal pane cloning is unavailable")
	}
	type completion struct {
		result bashResultT
		err    error
	}
	completed := make(chan completion, 1)
	command := "bash -c '" + strings.ReplaceAll(script, "'", "'\"'\"'") + "'"
	if err := cloner.ClonePane(command, func(output []byte, exitCode int, err error) {
		result := bashResultT{Output: string(output)}
		if err == nil {
			result.ExitCode = &exitCode
		}
		select {
		case completed <- completion{result: result, err: err}:
		default:
		}
	}); err != nil {
		return bashResultT{}, fmt.Errorf("create Bash terminal pane: %w", err)
	}
	select {
	case finished := <-completed:
		return finished.result, finished.err
	case <-ctx.Done():
		return bashResultT{}, fmt.Errorf("stopped waiting for Bash: %w; the cloned command may still be running", ctx.Err())
	}
}

func (t *Bash) ObservationStructured(input any, output string, err error) aitypes.ToolObservation {
	request, ok := input.(*bashInputT)
	if !ok || request == nil {
		return aitypes.ToolObservation{Tool: t.Name(), Status: "error", Error: fmt.Sprintf("invalid structured input %T", input)}
	}
	observation := aitypes.ToolObservation{
		Tool:   t.Name(),
		Status: "ok",
	}
	if err != nil {
		observation.Status = "error"
		observation.Error = err.Error()
	}
	command := aitypes.CommandObservation{
		Command: request.Script,
		Status:  observation.Status,
	}
	var result bashResultT
	if json.Unmarshal([]byte(output), &result) == nil {
		command.Summary = firstNonEmptyLine(result.Output)
		command.ExitCode = result.ExitCode
		if result.Error != "" {
			observation.Status = "error"
			observation.Error = result.Error
			command.Status = "error"
		} else if result.ExitCode != nil && *result.ExitCode != 0 {
			observation.Status = "error"
			command.Status = "failed"
		}
	}
	observation.CommandsRun = []aitypes.CommandObservation{command}
	return observation
}

func firstNonEmptyLine(output string) string {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			return line
		}
	}
	return ""
}
