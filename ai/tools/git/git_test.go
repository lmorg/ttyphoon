package gittool

import (
	"context"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
	"github.com/lmorg/ttyphoon/types"
)

type gitToolTestRenderer struct {
	types.Renderer
	command string
	output  []byte
	exit    int
}

func (renderer *gitToolTestRenderer) ClonePane(command string, complete func([]byte, int, error)) error {
	renderer.command = command
	complete(renderer.output, renderer.exit, nil)
	return nil
}

type gitToolTestAgent struct {
	root     string
	renderer types.Renderer
}

func (agent *gitToolTestAgent) Renderer() types.Renderer                      { return agent.renderer }
func (agent *gitToolTestAgent) ServiceName() string                           { return "test" }
func (agent *gitToolTestAgent) GetMeta() *aitypes.Meta                        { return nil }
func (agent *gitToolTestAgent) ProjectRoot() string                           { return agent.root }
func (agent *gitToolTestAgent) EnvironmentValue(string) string                { return "" }
func (agent *gitToolTestAgent) ImageGenerationEnvironmentValue(string) string { return "" }

func TestGitToolEnabledAndRequiresInvocationPermission(t *testing.T) {
	tool, err := (&Git{}).New(nil)
	if err != nil {
		t.Fatal(err)
	}
	if !tool.Enabled() {
		t.Fatal("Git tool is disabled by default")
	}
	permissions := tool.(*Git).DefaultPermissions()
	if permissions.Invocation != "askPermission" {
		t.Fatalf("default invocation permission = %q, want askPermission", permissions.Invocation)
	}
	if permissions.Subagents != "deny" {
		t.Fatalf("default subagent permission = %q, want deny", permissions.Subagents)
	}
}

func TestGitToolRunsReadOnlyActionThroughClonePane(t *testing.T) {
	gitPath, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git is not installed")
	}
	root := t.TempDir()
	for _, args := range [][]string{{"init", "--quiet", "--initial-branch=main"}} {
		command := exec.Command(gitPath, append([]string{"-C", root}, args...)...)
		if output, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, output)
		}
	}
	renderer := &gitToolTestRenderer{output: []byte("## main\n M file.txt\n")}
	tool := &Git{agent: &gitToolTestAgent{root: root, renderer: renderer}, enabled: true}
	response, err := tool.CallStructured(context.Background(), &gitInputT{Action: "status"})
	if err != nil {
		t.Fatal(err)
	}
	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	wantCommand := "git -C " + quoteGitTestArg(canonicalRoot) + " '--no-pager' 'status' '--short' '--branch'"
	if renderer.command != wantCommand {
		t.Fatalf("ClonePane command = %q, want %q", renderer.command, wantCommand)
	}
	var result gitResultT
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		t.Fatal(err)
	}
	if result.Action != "status" || result.Output != string(renderer.output) || result.Error != "" {
		t.Fatalf("Git result = %+v", result)
	}
}

func quoteGitTestArg(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func TestGitToolRejectsMutatingActionsBeforeClonePane(t *testing.T) {
	renderer := &gitToolTestRenderer{}
	tool := &Git{agent: &gitToolTestAgent{root: "/does/not/matter", renderer: renderer}, enabled: true}
	for _, action := range []string{"reset", "push", "commit", "fetch", "push-tag"} {
		response, err := tool.CallStructured(context.Background(), &gitInputT{Action: action})
		if err != nil {
			t.Fatal(err)
		}
		var result gitResultT
		if err := json.Unmarshal([]byte(response), &result); err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(result.Error, "unsupported read-only Git action") {
			t.Fatalf("action %q error = %q, want unsupported read-only action", action, result.Error)
		}
		if renderer.command != "" {
			t.Fatalf("ClonePane was called for non-read-only action %q: %q", action, renderer.command)
		}
	}
}
