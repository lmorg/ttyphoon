package workspace

import "testing"

func TestWorkspaceMergesDefaultAndWindowEnvironment(t *testing.T) {
	settings := &WorkspaceT{
		Defaults: TermT{
			Pwd:  "/default",
			Cmd:  "default-command",
			Envs: map[string]string{"SHARED": "default", "DEFAULT_ONLY": "value"},
		},
		Terminals: map[string]TermT{
			"project": {
				Pwd:  "/project",
				Cmd:  "project-command",
				Envs: map[string]string{"SHARED": "project", "PROJECT_ONLY": "value"},
			},
		},
	}

	got := settings.Workspace("project")
	if got.Pwd != "/project" || got.Cmd != "project-command" {
		t.Fatalf("workspace settings = %+v", got)
	}
	if got.Envs["SHARED"] != "project" || got.Envs["DEFAULT_ONLY"] != "value" || got.Envs["PROJECT_ONLY"] != "value" {
		t.Fatalf("workspace environment = %#v", got.Envs)
	}
}
