package graphtools

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
	"github.com/lmorg/ttyphoon/utils/file"
	"github.com/lmorg/ttyphoon/utils/graphs"
)

type CreateGraph struct {
	enabled bool
}

func init() {
	agent.ToolsAdd(&CreateGraph{})
}

//go:embed description.md
var description string

func (t *CreateGraph) New(_ aitypes.Agent) (aitypes.Tool, error) {
	return &CreateGraph{enabled: true}, nil
}

func (t *CreateGraph) Enabled() bool       { return t.enabled }
func (t *CreateGraph) Toggle()             { t.enabled = !t.enabled }
func (t *CreateGraph) Name() string        { return "createGraph" }
func (t *CreateGraph) Path() string        { return "internal" }
func (t *CreateGraph) Description() string { return description }
func (t *CreateGraph) DefaultPermissions() aitypes.DefaultPermissions {
	return aitypes.DefaultPermissions{Invocation: "alwaysAllow", Subagents: "deny"}
}

func (t *CreateGraph) InputType() reflect.Type { return reflect.TypeOf(createGraphInputT{}) }

type createGraphInputT struct {
	Headings          []string   `json:"headings"`
	Rows              [][]string `json:"rows"`
	Type              string     `json:"type"`
	Title             string     `json:"title,omitempty"`
	CategoryColumn    int        `json:"categoryColumn"`
	ValueColumns      []int      `json:"valueColumns"`
	SeriesNames       []string   `json:"seriesNames,omitempty"`
	CategoryAxisTitle string     `json:"categoryAxisTitle,omitempty"`
	ValueAxisTitle    string     `json:"valueAxisTitle,omitempty"`
}

func (t *CreateGraph) CallStructured(ctx context.Context, input any) (string, error) {
	request, ok := input.(*createGraphInputT)
	if !ok || request == nil {
		return "", fmt.Errorf("createGraph received an invalid structured input %T", input)
	}
	return t.create(ctx, request, "")
}

func (t *CreateGraph) create(_ context.Context, request *createGraphInputT, outputDir string) (string, error) {
	const format = graphs.PNG
	data, err := graphs.RenderTable(graphs.Table{Headings: request.Headings, Rows: request.Rows}, graphs.Spec{
		Type:              graphs.Type(strings.ToLower(strings.TrimSpace(request.Type))),
		Format:            format,
		Title:             request.Title,
		CategoryColumn:    request.CategoryColumn,
		ValueColumns:      request.ValueColumns,
		SeriesNames:       request.SeriesNames,
		CategoryAxisTitle: request.CategoryAxisTitle,
		ValueAxisTitle:    request.ValueAxisTitle,
	})
	if err != nil {
		return "", err
	}
	if outputDir == "" {
		outputDir, err = defaultGraphImagesDir()
		if err != nil {
			return "", fmt.Errorf("resolve chart output directory: %w", err)
		}
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return "", fmt.Errorf("create chart output directory: %w", err)
	}
	file, err := os.CreateTemp(outputDir, "generated-chart-*"+"."+string(format))
	if err != nil {
		return "", fmt.Errorf("create chart output file: %w", err)
	}
	path := file.Name()
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		_ = os.Remove(path)
		return "", fmt.Errorf("write chart output: %w", err)
	}
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return "", fmt.Errorf("close chart output: %w", err)
	}
	return fmt.Sprintf("Chart saved to: %s\n\n![%s](%s)", path, markdownAlt(request.Title), filepath.ToSlash(path)), nil
}

func defaultGraphImagesDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(file.DocumentsDir(home), ".images"), nil
}

func markdownAlt(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return "chart"
	}
	text = strings.ReplaceAll(text, "\\", "\\\\")
	return strings.ReplaceAll(text, "]", "\\]")
}

func (t *CreateGraph) ObservationStructured(input any, output string, err error) aitypes.ToolObservation {
	request, ok := input.(*createGraphInputT)
	if !ok || request == nil {
		return aitypes.ToolObservation{Tool: t.Name(), Status: "error", Error: fmt.Sprintf("invalid structured input %T", input)}
	}
	observation := aitypes.ToolObservation{Tool: t.Name(), Status: "ok", Counts: map[string]int{"charts": 1}}
	if err != nil {
		observation.Status = "error"
		observation.Error = err.Error()
		observation.Counts["charts"] = 0
	}
	if outputPath := graphPathFromOutput(output); outputPath != "" {
		observation.Outputs = []string{outputPath}
	}
	if strings.TrimSpace(request.Title) != "" {
		observation.Summary = request.Title
	}
	return observation
}

func graphPathFromOutput(output string) string {
	const prefix = "Chart saved to: "
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(line, prefix))
		}
	}
	return ""
}
