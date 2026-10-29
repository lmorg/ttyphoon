package gittool

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
	gitutils "github.com/lmorg/ttyphoon/utils/git"
	"github.com/lmorg/ttyphoon/utils/markdownutil"
)

const maxGitOutputRunes = 32000

var readOnlyActions = map[string]bool{
	"status":   true,
	"diff":     true,
	"log":      true,
	"show":     true,
	"branches": true,
	"remotes":  true,
}

type Git struct {
	agent   aitypes.Agent
	enabled bool
}

func init() {
	agent.ToolsAdd(&Git{})
}

//go:embed description.md
var description string

func (t *Git) New(agt aitypes.Agent) (aitypes.Tool, error) {
	return &Git{agent: agt, enabled: true}, nil
}

func (t *Git) Enabled() bool { return t.enabled }
func (t *Git) Toggle()       { t.enabled = !t.enabled }
func (t *Git) Name() string  { return "git" }
func (t *Git) Path() string  { return "internal" }
func (t *Git) Description() string {
	return strings.TrimSpace(description)
}
func (t *Git) DefaultPermissions() aitypes.DefaultPermissions {
	return aitypes.DefaultPermissions{Invocation: "askPermission", Subagents: "deny"}
}

type gitInputT struct {
	Action   string `json:"action"`
	Path     string `json:"path,omitempty"`
	Staged   bool   `json:"staged,omitempty"`
	Revision string `json:"revision,omitempty"`
	Limit    int    `json:"limit,omitempty"`
}

type gitResultT struct {
	Action string `json:"action"`
	Output string `json:"output,omitempty"`
	Error  string `json:"error,omitempty"`
}

type gitPaneCloner interface {
	ClonePane(string, func([]byte, int, error)) error
}

func (t *Git) InputType() reflect.Type { return reflect.TypeOf(gitInputT{}) }

func (t *Git) CallStructured(ctx context.Context, input any) (string, error) {
	request, ok := input.(*gitInputT)
	if !ok || request == nil {
		return "", fmt.Errorf("git received an invalid structured input %T", input)
	}
	request.Action = strings.ToLower(strings.TrimSpace(request.Action))
	if !readOnlyActions[request.Action] {
		encoded, err := json.Marshal(gitResultT{
			Action: request.Action,
			Error:  "unsupported read-only Git action; use status, diff, log, show, branches, or remotes",
		})
		return string(encoded), err
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if t.agent == nil || t.agent.Renderer() == nil {
		return "", fmt.Errorf("Git terminal inspection is unavailable")
	}
	cloner, ok := t.agent.Renderer().(gitPaneCloner)
	if !ok {
		return "", fmt.Errorf("Git terminal inspection requires ClonePane")
	}
	parameters := make(map[string]string, 4)
	if request.Path != "" {
		parameters["path"] = request.Path
	}
	if request.Staged {
		parameters["staged"] = strconv.FormatBool(request.Staged)
	}
	if request.Revision != "" {
		parameters["revision"] = request.Revision
	}
	if request.Limit != 0 {
		parameters["limit"] = strconv.Itoa(request.Limit)
	}

	result := gitutils.RunActionContext(ctx, t.agent.ProjectRoot(), gitutils.ActionRequest{
		Name:       request.Action,
		Parameters: parameters,
	}, nil, func(command string, complete func(gitutils.InteractiveActionResult)) error {
		return cloner.ClonePane(command, func(output []byte, exitCode int, err error) {
			complete(gitutils.InteractiveActionResult{
				Output:   string(output),
				ExitCode: exitCode,
				Err:      err,
			})
		})
	})
	response := gitResultT{
		Action: request.Action,
		Output: boundOutput(result["stdout"]),
		Error:  result["error"],
	}
	encoded, err := json.Marshal(response)
	return string(encoded), err
}

func (t *Git) CallDescription(_ context.Context, input any) string {
	request, ok := input.(*gitInputT)
	if !ok || request == nil {
		return "Read-only Git inspection"
	}
	description := "Read-only Git action: " + markdownutil.EscapeMarkdownText(request.Action)
	if request.Path != "" {
		description += "\nPath: " + markdownutil.EscapeMarkdownText(request.Path)
	}
	if request.Revision != "" {
		description += "\nRevision: " + markdownutil.EscapeMarkdownText(request.Revision)
	}
	return description
}

func (t *Git) ObservationStructured(input any, output string, err error) aitypes.ToolObservation {
	request, ok := input.(*gitInputT)
	if !ok || request == nil {
		return aitypes.ToolObservation{Tool: t.Name(), Status: "error", Error: fmt.Sprintf("invalid structured input %T", input)}
	}
	observation := aitypes.ToolObservation{
		Tool:   t.Name(),
		Status: "ok",
		CommandsRun: []aitypes.CommandObservation{{
			Command: "git " + request.Action,
			Status:  "ok",
		}},
	}
	var result gitResultT
	if err := json.Unmarshal([]byte(output), &result); err == nil {
		if summary := firstOutputLine(result.Output); summary != "" {
			observation.CommandsRun[0].Summary = summary
		}
		if result.Error != "" {
			observation.Status = "error"
			observation.Error = result.Error
			observation.CommandsRun[0].Status = "failed"
		}
	}
	if err != nil {
		observation.Status = "error"
		observation.Error = err.Error()
		observation.CommandsRun[0].Status = "error"
	}
	return observation
}

func boundOutput(output string) string {
	runes := []rune(output)
	if len(runes) <= maxGitOutputRunes {
		return output
	}
	return string(runes[:maxGitOutputRunes]) + "\n[Git output truncated]"
}

func firstOutputLine(output string) string {
	for _, line := range strings.Split(output, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			return line
		}
	}
	return ""
}
