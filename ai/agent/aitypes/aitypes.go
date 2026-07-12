package aitypes

import (
	"context"

	"github.com/lmorg/ttyphoon/types"
)

type Agent interface {
	Renderer() types.Renderer
	ServiceName() string
	GetMeta() *Meta
}

type Tool interface {
	New(Agent) (Tool, error)
	Enabled() bool
	Toggle()
	Name() string
	Path() string
	Description() string
	Call(context.Context, string) (string, error)
}

type Meta struct {
	CmdLine     string
	Pwd         string
	OutputBlock string
	Function    string
	Variables   map[string]any
}
