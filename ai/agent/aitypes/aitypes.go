package aitypes

import (
	"context"
	"reflect"

	"github.com/lmorg/ttyphoon/types"
)

type Agent interface {
	Renderer() types.Renderer
	ServiceName() string
	GetMeta() *Meta
	ProjectRoot() string
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

// StructuredTool opts a native tool into a schema derived from its Go JSON
// input type (object, array, or scalar). The runtime decodes the model's JSON
// value once and passes a pointer to that value to CallStructured. Tools not
// implementing this interface keep the legacy string-input contract.
type StructuredTool interface {
	Tool
	InputType() reflect.Type
	CallStructured(context.Context, any) (string, error)
}

// StructuredToolObservationProvider consumes the already-decoded input so
// structured tools do not need to unmarshal their arguments again for logging.
type StructuredToolObservationProvider interface {
	ObservationStructured(input any, output string, err error) ToolObservation
}

type ToolObservationProvider interface {
	Observation(input, output string, err error) ToolObservation
}

type ToolObservation struct {
	Tool              string
	Status            string
	Summary           string
	Inputs            []string
	Outputs           []string
	FilesRead         []string
	FilesModified     []string
	DirectoriesListed []string
	SearchesRun       []SearchObservation
	CommandsRun       []CommandObservation
	Counts            map[string]int
	Error             string
}

type SearchObservation struct {
	Query       string
	FileFilter  string
	ResultCount int
	TopPaths    []string
}

type CommandObservation struct {
	Command  string
	ExitCode *int
	Status   string
	Summary  string
}

type DefaultPermissions struct {
	Invocation string
	Subagents  string
}

type Meta struct {
	CmdLine     string
	Pwd         string // this is not the same as the application working directory
	OutputBlock string
	Function    string
	Variables   map[string]any
}

// ImageAttachment carries an inline image to send alongside a prompt.
type ImageAttachment struct {
	MIMEType string
	Base64   string
}
