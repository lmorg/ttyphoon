package workspace

import (
	"reflect"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestWorkspaceMergesDefaultAndWindowEnvironment(t *testing.T) {
	settings := &WorkspaceT{
		Defaults: TermT{
			Pwd:      "/default",
			Cmd:      "default-command",
			EnvFiles: []string{"default.env"},
			Envs:     map[string]string{"SHARED": "default", "DEFAULT_ONLY": "value"},
		},
		Terminals: map[string]TermT{
			"project": {
				Pwd:      "/project",
				Cmd:      "project-command",
				EnvFiles: []string{"project.env"},
				Envs:     map[string]string{"SHARED": "project", "PROJECT_ONLY": "value"},
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
	if !reflect.DeepEqual(got.EnvFiles, []string{"default.env", "project.env"}) {
		t.Fatalf("workspace environment files = %#v", got.EnvFiles)
	}
	got.EnvFiles[0] = "changed"
	if settings.Defaults.EnvFiles[0] != "default.env" {
		t.Fatal("workspace environment files modified defaults")
	}
	defaults := settings.Workspace("unconfigured")
	if defaults.Pwd != "/default" || defaults.Cmd != "default-command" || defaults.Envs["DEFAULT_ONLY"] != "value" {
		t.Fatalf("unconfigured workspace did not retain defaults: %+v", defaults)
	}
}

func TestWorkspaceEnvFilesDecodeFromYAML(t *testing.T) {
	var settings WorkspaceT
	if err := yaml.Unmarshal([]byte("Defaults:\n  EnvFiles: [default.env]\nTerminals:\n  project:\n    EnvFiles: [/tmp/project.env]\n"), &settings); err != nil {
		t.Fatal(err)
	}
	if got := settings.Workspace("project").EnvFiles; !reflect.DeepEqual(got, []string{"default.env", "/tmp/project.env"}) {
		t.Fatalf("workspace environment files = %#v", got)
	}
}
