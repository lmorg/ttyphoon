package aitypes

import (
	"context"

	"github.com/lmorg/ttyphoon/types"
)

type Agent interface {
	Renderer() types.Renderer
	ServiceName() string
	GetMeta() *Meta
	// EnvironmentValue resolves a service-scoped value, falling back to the process environment.
	EnvironmentValue(string) string
	// ImageGenerationEnvironmentValue resolves image-generation settings for the configured service.
	ImageGenerationEnvironmentValue(string) string
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

// ImageAttachment carries an inline image to send alongside a prompt.
type ImageAttachment struct {
	MIMEType string
	Base64   string
}
