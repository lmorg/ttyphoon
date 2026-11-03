package workspace

import "maps"

type WorkspaceT struct {
	Defaults  TermT            `yaml:"Defaults"`
	Terminals map[string]TermT `yaml:"Terminals"`
}

type TermT struct {
	Pwd  string            `yaml:"Pwd"`
	Envs map[string]string `yaml:"Envs"`
	Cmd  string            `yaml:"Cmd"`
}

// Workspace will never return nil. It is used to offer the workspace values, plus defaults should the workspace not exist or workspace specific settings are undefined
func (w *WorkspaceT) Workspace(name string) *TermT {
	t := &TermT{
		Pwd:  w.Defaults.Pwd,
		Envs: make(map[string]string),
		Cmd:  w.Defaults.Cmd,
	}
	maps.Copy(t.Envs, w.Defaults.Envs)

	if terminal, ok := w.Terminals[name]; ok {
		if terminal.Pwd != "" {
			t.Pwd = terminal.Pwd
		}
		if terminal.Cmd != "" {
			t.Cmd = terminal.Cmd
		}
		maps.Copy(t.Envs, terminal.Envs)
	}

	return t
}
