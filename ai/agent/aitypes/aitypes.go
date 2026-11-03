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
}

// StringInputTool is the compatibility contract for tools that consume JSON
// arguments as a string instead of a decoded value.
type StringInputTool interface {
	Tool
	Call(context.Context, string) (string, error)
}

// JSONSchemaTool combines a declared JSON Schema with the string-input
// compatibility adapter, for tools such as MCP tools.
type JSONSchemaTool interface {
	StringInputTool
	JSONSchema() []byte
}

// StructuredTool opts a native tool into a schema derived from its Go JSON
// input type (object, array, or scalar). The runtime decodes the model's JSON
// value once and passes a pointer to that value to CallStructured.
type StructuredTool interface {
	Tool
	InputType() reflect.Type
	CallStructured(context.Context, any) (string, error)
}

// ToolCallDescriber optionally provides Markdown describing a pending tool
// invocation. input matches the value passed to Call or CallStructured.
type ToolCallDescriber interface {
	CallDescription(context.Context, any) string
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
	CmdLine        string
	Pwd            string // this is not the same as the application working directory
	OutputBlock    string
	Function       string
	Variables      map[string]any
	SkillResources map[string]string
}

// ImageAttachment carries an inline image to send alongside a prompt.
type ImageAttachment = types.ImageAttachment
