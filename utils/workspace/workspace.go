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

func (w *WorkspaceT) Workspace(name string) *TermT {
	t := &TermT{
		Pwd: w.Defaults.Pwd,
		Cmd: w.Defaults.Cmd,
	}
	maps.Copy(t.Envs, w.Defaults.Envs)

	t.Pwd = w.Terminals[name].Pwd
	t.Cmd = w.Terminals[name].Cmd
	maps.Copy(t.Envs, w.Terminals[name].Envs)

	return t
}
