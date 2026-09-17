package shell

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
	"github.com/lmorg/ttyphoon/debug"
	"github.com/lmorg/ttyphoon/types"
)

type CommandLine struct {
	agent   aitypes.Agent
	enabled bool
	term    types.Term
}

func init() {
	agent.ToolsAdd(&CommandLine{})
}

//go:embed cmdline_description.md
var cmdlineDescription string

func (t *CommandLine) New(agent aitypes.Agent) (aitypes.Tool, error) {
	return &CommandLine{agent: agent, enabled: false}, nil
}

func (t *CommandLine) Enabled() bool { return t.enabled }
func (t *CommandLine) Toggle()       { t.enabled = !t.enabled }

func (t *CommandLine) Name() string { return "commandLine" }
func (t *CommandLine) Path() string { return "internal" }
func (t *CommandLine) Description() string {
	t.term = t.agent.Renderer().ActiveTile().GetTerm()
	return cmdlineDescription + "\n$SHELL=" + t.shellName()
}
func (t *CommandLine) DefaultPermissions() aitypes.DefaultPermissions {
	return aitypes.DefaultPermissions{Invocation: "askPermission", Subagents: "deny"}
}

func (t *CommandLine) shellName() string {
	return t.term.GetEnvVars()["SHELL"]
}

func (t *CommandLine) Call(ctx context.Context, input string) (string, error) {
	debug.Log(input)

	c := make(chan *types.BlockCallbackT)

	t.term.SetCommandCallback(func(cb *types.BlockCallbackT) {
		c <- cb
	})

	bct := <-c

	t.term.Reply([]byte(input))

	result := &resultT{
		Output:    bct.Output,
		ExitCode:  bct.Meta.ExitNum,
		TimeStart: bct.Meta.TimeStart.String(),
		TimeEnd:   bct.Meta.TimeEnd.String(),
		Duration:  bct.Meta.TimeEnd.Sub(bct.Meta.TimeStart).String(),
	}

	b, err := json.Marshal(result)
	if err != nil {
		return fmt.Sprintf("Command ran but cannot marshall output: %v", err), nil
	}
	return string(b), nil
}

func (t *CommandLine) Observation(input, output string, err error) aitypes.ToolObservation {
	observation := aitypes.ToolObservation{Tool: t.Name(), Status: "ok"}
	if err != nil {
		observation.Status = "error"
		observation.Error = err.Error()
	}

	command := aitypes.CommandObservation{Command: input, Status: observation.Status}
	var result resultT
	if json.Unmarshal([]byte(output), &result) == nil {
		command.ExitCode = &result.ExitCode
		if result.ExitCode != 0 {
			command.Status = "failed"
			observation.Status = "error"
		}
		command.Summary = firstNonEmptyLine(result.Output)
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

type resultT struct {
	Output    string
	ExitCode  int
	TimeStart string
	TimeEnd   string
	Duration  string
}
