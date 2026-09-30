package shell

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
)

func TestShellToolsExposeStructuredInputTypes(t *testing.T) {
	var commandLine aitypes.StructuredTool = &CommandLine{}
	if got := commandLine.InputType(); got != reflect.TypeOf(commandLineInputT{}) {
		t.Fatalf("commandLine InputType() = %v, want commandLineInputT", got)
	}

	var envVars aitypes.StructuredTool = &EnvVars{}
	if got := envVars.InputType(); got != reflect.TypeOf(envVarsInputT{}) {
		t.Fatalf("envVars InputType() = %v, want empty object", got)
	}
}

func TestCommandLineObservationRecordsCommandRun(t *testing.T) {
	output, err := json.Marshal(resultT{Output: "\nall good\nmore detail", ExitCode: 0})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	observation := (&CommandLine{}).Observation("go test ./...", string(output), nil)
	if observation.Tool != "commandLine" || observation.Status != "ok" {
		t.Fatalf("observation = %+v, want commandLine ok", observation)
	}
	if len(observation.CommandsRun) != 1 {
		t.Fatalf("CommandsRun = %d, want 1", len(observation.CommandsRun))
	}
	command := observation.CommandsRun[0]
	if command.Command != "go test ./..." || command.Status != "ok" || command.ExitCode == nil || *command.ExitCode != 0 || command.Summary != "all good" {
		t.Fatalf("command observation = %+v", command)
	}
}

func TestCommandLineObservationMarksNonZeroExitFailed(t *testing.T) {
	output, err := json.Marshal(resultT{Output: "failure", ExitCode: 2})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	observation := (&CommandLine{}).Observation("go test ./...", string(output), nil)
	if observation.Status != "error" {
		t.Fatalf("Status = %q, want error", observation.Status)
	}
	if observation.CommandsRun[0].Status != "failed" || observation.CommandsRun[0].ExitCode == nil || *observation.CommandsRun[0].ExitCode != 2 {
		t.Fatalf("command observation = %+v", observation.CommandsRun[0])
	}
}

func TestCommandLineStructuredObservationUsesDecodedCommand(t *testing.T) {
	command := commandLineInputT{Command: "go test ./..."}
	output, err := json.Marshal(resultT{Output: "all good", ExitCode: 0})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	observation := (&CommandLine{}).ObservationStructured(&command, string(output), nil)
	if len(observation.CommandsRun) != 1 || observation.CommandsRun[0].Command != command.Command {
		t.Fatalf("CommandsRun = %+v, want command %q", observation.CommandsRun, command.Command)
	}
}
