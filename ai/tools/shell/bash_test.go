package shell

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"reflect"
	"strings"
	"testing"

	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
	"github.com/lmorg/ttyphoon/types"
)

type bashTestAgent struct {
	aitypes.Agent
	renderer types.Renderer
}

func (agt *bashTestAgent) Renderer() types.Renderer { return agt.renderer }

type bashTestRenderer struct {
	types.Renderer
	clone func(string, func([]byte, int, error)) error
}

func (renderer *bashTestRenderer) ClonePane(command string, callback func([]byte, int, error)) error {
	return renderer.clone(command, callback)
}

func TestBashExposesStructuredInput(t *testing.T) {
	tool := &Bash{}
	var structured aitypes.StructuredTool = tool
	if got := structured.InputType(); got != reflect.TypeOf(bashInputT{}) {
		t.Fatalf("InputType() = %v, want bashInputT", got)
	}
}

func TestBashApprovalDescriptionIncludesOneLineSummaryAndExactScript(t *testing.T) {
	var request bashInputT
	if err := json.Unmarshal([]byte(`{"summary":"Run the test suite","script":"go test ./...\nprintf 'done\\n'"}`), &request); err != nil {
		t.Fatal(err)
	}
	description := (&Bash{}).CallDescription(context.Background(), &request)
	want := "Run the test suite\n\n```bash\ngo test ./...\nprintf 'done\\n'\n```"
	if description != want {
		t.Fatalf("approval description = %q, want %q", description, want)
	}
}

func TestBashApprovalSummaryIsSingleLineAndEscaped(t *testing.T) {
	request := &bashInputT{Summary: "  Check\n [sources](ttyphoon://example)\t<script>  ", Script: "pwd"}
	description := (&Bash{}).CallDescription(context.Background(), request)
	want := "Check \\[sources\\]\\(ttyphoon://example\\) &lt;script&gt;\n\n```bash\npwd\n```"
	if description != want {
		t.Fatalf("approval description = %q, want %q", description, want)
	}
	request.Summary = ""
	if got := (&Bash{}).CallDescription(context.Background(), request); got != "```bash\npwd\n```" {
		t.Fatalf("empty-summary description = %q", got)
	}
}

func TestBashRunsScriptInClonedPaneAndReturnsScreenOutput(t *testing.T) {
	renderer := &bashTestRenderer{clone: func(command string, callback func([]byte, int, error)) error {
		if !strings.HasPrefix(command, "bash -c '") || !strings.Contains(command, "stdout") {
			t.Fatalf("clone command = %q", command)
		}
		if strings.Contains(command, "purpose-only-token") {
			t.Fatal("approval summary was included in the executed command")
		}
		callback([]byte("stdout\nstderr\n"), 0, nil)
		return nil
	}}
	tool := &Bash{agent: &bashTestAgent{renderer: renderer}}
	output, err := tool.CallStructured(context.Background(), &bashInputT{
		Script:  "printf 'stdout\\n'; printf 'stderr\\n' >&2",
		Summary: "purpose-only-token",
	})
	if err != nil {
		t.Fatalf("CallStructured() error = %v", err)
	}
	var result bashResultT
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatal(err)
	}
	if result.Output != "stdout\nstderr\n" || result.ExitCode == nil || *result.ExitCode != 0 {
		t.Fatalf("result = %+v", result)
	}
}

func TestBashReturnsRecoverableExitFailureAsOutput(t *testing.T) {
	renderer := &bashTestRenderer{clone: func(_ string, callback func([]byte, int, error)) error {
		callback([]byte("before failure\n"), 7, nil)
		return nil
	}}
	output, err := (&Bash{agent: &bashTestAgent{renderer: renderer}}).CallStructured(context.Background(), &bashInputT{
		Script: "printf 'before failure\\n'; exit 7",
	})
	if err != nil {
		t.Fatalf("CallStructured() error = %v, want recoverable failure as output", err)
	}
	var result bashResultT
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatal(err)
	}
	if result.Output != "before failure\n" || result.ExitCode == nil || *result.ExitCode != 7 {
		t.Fatalf("result = %+v, want partial output and exit code 7", result)
	}
}

func TestBashPreservesScriptQuoting(t *testing.T) {
	for _, script := range []string{
		"printf '%s' \"single ' and double \\\" quotes\"",
		"value='two words'\nprintf '%s' \"$value\"",
		"printf '%s' '$(must-not-run); literal $HOME'",
	} {
		t.Run(script, func(t *testing.T) {
			want, err := exec.Command("bash", "-c", script).CombinedOutput()
			if err != nil {
				t.Fatal(err)
			}
			renderer := &bashTestRenderer{clone: func(command string, callback func([]byte, int, error)) error {
				got, err := exec.Command("bash", "-c", command).CombinedOutput()
				if err != nil {
					return err
				}
				callback(got, 0, nil)
				return nil
			}}
			result, err := (&Bash{agent: &bashTestAgent{renderer: renderer}}).run(context.Background(), script)
			if err != nil || result.Output != string(want) {
				t.Fatalf("script output = %q, %v; want %q", result.Output, err, want)
			}
		})
	}
}

func TestBashReturnsCloneAndCallbackFailures(t *testing.T) {
	for _, callbackFailure := range []bool{false, true} {
		t.Run(fmt.Sprintf("callback=%t", callbackFailure), func(t *testing.T) {
			renderer := &bashTestRenderer{clone: func(_ string, callback func([]byte, int, error)) error {
				if callbackFailure {
					callback([]byte("partial output"), 0, fmt.Errorf("synthetic pane closed"))
					return nil
				}
				return fmt.Errorf("synthetic clone failed")
			}}
			tool := &Bash{agent: &bashTestAgent{renderer: renderer}}
			output, err := tool.CallStructured(context.Background(), &bashInputT{Script: "printf test"})
			if err != nil {
				t.Fatal(err)
			}
			var result bashResultT
			if err := json.Unmarshal([]byte(output), &result); err != nil {
				t.Fatal(err)
			}
			if result.Error == "" || result.ExitCode != nil {
				t.Fatalf("failure result = %+v; want error without a success exit code", result)
			}
			if callbackFailure && result.Output != "partial output" {
				t.Fatalf("callback output lost: %+v", result)
			}
			if observation := tool.ObservationStructured(&bashInputT{Script: "printf test"}, output, nil); observation.Status != "error" {
				t.Fatalf("failure observation = %+v", observation)
			}
		})
	}
}

func TestBashCancellationStopsWaitingAndAllowsLateCompletion(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var complete func([]byte, int, error)
	renderer := &bashTestRenderer{clone: func(_ string, callback func([]byte, int, error)) error {
		complete = callback
		cancel()
		return nil
	}}
	result, err := (&Bash{agent: &bashTestAgent{renderer: renderer}}).run(ctx, "read answer")
	if err == nil || !strings.Contains(err.Error(), "may still be running") || result.ExitCode != nil {
		t.Fatalf("cancelled result = %+v, %v", result, err)
	}
	complete([]byte("late output"), 0, nil)
	complete([]byte("duplicate output"), 0, nil)
}

func TestBashRequiresCloneSupportAndHonorsPreCancelledContext(t *testing.T) {
	if _, err := (&Bash{}).run(context.Background(), "printf test"); err == nil {
		t.Fatal("tool silently fell back without a terminal cloner")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	tool := &Bash{agent: &bashTestAgent{renderer: &bashTestRenderer{clone: func(string, func([]byte, int, error)) error {
		t.Fatal("cancelled tool created a pane")
		return nil
	}}}}
	if _, err := tool.run(ctx, "printf test"); err != context.Canceled {
		t.Fatalf("cancelled run error = %v", err)
	}
}

func TestBashObservationRecordsScriptExitStatus(t *testing.T) {
	for _, exitCode := range []int{0, 7} {
		output, err := json.Marshal(bashResultT{Output: "\nall done\nmore detail", ExitCode: &exitCode})
		if err != nil {
			t.Fatal(err)
		}
		request := &bashInputT{Script: "go test ./..."}
		observation := (&Bash{}).ObservationStructured(request, string(output), nil)
		if len(observation.CommandsRun) != 1 {
			t.Fatalf("observation = %+v", observation)
		}
		command := observation.CommandsRun[0]
		if command.Command != request.Script || command.Summary != "all done" || command.ExitCode == nil || *command.ExitCode != exitCode {
			t.Fatalf("command observation = %+v", command)
		}
		if exitCode == 0 && (observation.Status != "ok" || command.Status != "ok") {
			t.Fatalf("successful observation = %+v", observation)
		}
		if exitCode != 0 && (observation.Status != "error" || command.Status != "failed") {
			t.Fatalf("failed observation = %+v", observation)
		}
	}
	var envVars aitypes.StructuredTool = &EnvVars{}
	if envVars.InputType() != reflect.TypeOf(envVarsInputT{}) {
		t.Fatal("envVars structured input changed")
	}
}
