package filetools

import (
	"bufio"
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"text/template"

	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/utils/markdownutil"
	"golang.org/x/tools/txtar"
)

type ReadFiles struct {
	agent   aitypes.Agent
	enabled bool
}

func init() {
	agent.ToolsAdd(&ReadFiles{})
}

//go:embed read_description.md
var readDescription string

//go:embed read_auth_desc.md
var readFileAuthDescription string

var readFileAuthDescriptionTemplate = template.Must(template.New("read-file-auth-description").Funcs(markdownutil.TemplateFuncMap()).Parse(readFileAuthDescription))

func (f *ReadFiles) New(agent aitypes.Agent) (aitypes.Tool, error) {
	return &ReadFiles{agent: agent, enabled: true}, nil
}

func (t *ReadFiles) Enabled() bool { return t.enabled }
func (t *ReadFiles) Toggle()       { t.enabled = !t.enabled }

func (t *ReadFiles) Name() string { return "readFiles" }
func (t *ReadFiles) Path() string { return "internal" }
func (t *ReadFiles) DefaultPermissions() aitypes.DefaultPermissions {
	return aitypes.DefaultPermissions{Invocation: "alwaysAllow", Subagents: "allow"}
}
func (t *ReadFiles) Description() string {
	return readDescription
}

type readFilesInputT struct {
	Files []readFileInputT `json:"files"`
}

type readFileInputT struct {
	Path      string `json:"path"`
	StartLine *int   `json:"startLine,omitempty"`
	EndLine   *int   `json:"endLine,omitempty"`
}

func (file *readFileInputT) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		return fmt.Errorf("each files entry must be a path string or an object with path and optional line bounds")
	}
	if data[0] == '"' {
		var path string
		if err := json.Unmarshal(data, &path); err != nil {
			return err
		}
		*file = readFileInputT{Path: path}
		return nil
	}
	type fileInput readFileInputT
	return json.Unmarshal(data, (*fileInput)(file))
}

func (request *readFilesInputT) UnmarshalJSON(data []byte) error {
	var decoded struct {
		Files     []readFileInputT `json:"files"`
		StartLine json.RawMessage  `json:"startLine"`
		EndLine   json.RawMessage  `json:"endLine"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	if len(decoded.StartLine) > 0 || len(decoded.EndLine) > 0 {
		return fmt.Errorf("startLine and endLine belong inside each files entry, not at the top level")
	}
	request.Files = decoded.Files
	return nil
}

func (file readFileInputT) lineRange() (int, int, error) {
	start, end := 1, 0
	if file.StartLine != nil {
		start = *file.StartLine
		if start < 1 {
			return 0, 0, fmt.Errorf("startLine must be at least 1")
		}
	}
	if file.EndLine != nil {
		end = *file.EndLine
		if end < start {
			return 0, 0, fmt.Errorf("endLine must be at least startLine (%d)", start)
		}
	}
	return start, end, nil
}

func (t *ReadFiles) InputType() reflect.Type { return reflect.TypeOf(readFilesInputT{}) }

func (t *ReadFiles) CallStructured(ctx context.Context, input any) (string, error) {
	request, ok := input.(*readFilesInputT)
	if !ok || request == nil {
		return "", fmt.Errorf("readFiles received an invalid structured input %T", input)
	}
	return t.callRequest(ctx, request)
}

func (t *ReadFiles) CallDescription(_ context.Context, input any) string {
	request, ok := input.(*readFilesInputT)
	if !ok || request == nil {
		return "Tool received an invalid structured input"
	}

	var description bytes.Buffer
	if err := readFileAuthDescriptionTemplate.Execute(&description, request); err != nil {
		return fmt.Sprintf("Unable to render description: %v", err)
	}
	return description.String()
}

func (t *ReadFiles) callFiles(ctx context.Context, files []string) (string, error) {
	request := &readFilesInputT{Files: make([]readFileInputT, len(files))}
	for index, path := range files {
		request.Files[index] = readFileInputT{Path: path}
	}
	return t.callRequest(ctx, request)
}

func (t *ReadFiles) callRequest(ctx context.Context, request *readFilesInputT) (response string, err error) {
	files := request.Files
	var archive txtar.Archive

	for i := range files {
		file := files[i]
		start, end, rangeErr := file.lineRange()
		if rangeErr != nil {
			return "", fmt.Errorf("%s: %w", file.Path, rangeErr)
		}
		filename, pathErr := resolveWorkspacePath(t.agent, file.Path)
		if pathErr != nil {
			archive.Files = append(archive.Files, txtar.File{
				Name: file.Path,
				Data: fmt.Appendf(nil, "!!! Cannot open file: %v", pathErr),
			})
			continue
		}

		t.agent.Renderer().DisplayNotification(types.NOTIFY_INFO, t.agent.ServiceName()+" requesting file: "+filename)

		var b []byte
		info, err := os.Stat(filename)
		if err != nil {
			b = fmt.Appendf(nil, "!!! Cannot open file: %v", err)

		} else if info.Name()[0] == '.' {
			b = fmt.Appendf(nil, "!!! Cannot open file: %s", "file hidden")

		} else {
			if file.StartLine == nil && file.EndLine == nil {
				b, err = os.ReadFile(filename)
			} else {
				b, err = readFileLineRange(ctx, filename, start, end)
			}
			if err != nil {
				b = fmt.Appendf(nil, "!!! Cannot open file: %v", err)
			}
		}

		archive.Files = append(archive.Files, txtar.File{
			Name: file.Path,
			Data: b,
		})

		err = nil
	}

	response = string(txtar.Format(&archive))

	return response, nil
}

func readFileLineRange(ctx context.Context, filename string, start, end int) ([]byte, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return readLineRange(ctx, file, start, end)
}

func readLineRange(ctx context.Context, source io.Reader, start, end int) ([]byte, error) {
	reader := bufio.NewReader(source)
	var output bytes.Buffer
	for line := 1; end == 0 || line <= end; {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		fragment, err := reader.ReadSlice('\n')
		if line >= start {
			output.Write(fragment)
		}
		if err == bufio.ErrBufferFull {
			continue
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		line++
	}
	return output.Bytes(), nil
}

func (t *ReadFiles) Observation(input, output string, err error) aitypes.ToolObservation {
	var files []string
	if json.Unmarshal([]byte(input), &files) != nil {
		return aitypes.ToolObservation{Tool: t.Name(), Status: "ok", Inputs: []string{input}}
	}
	return t.observation(files, err)
}

func (t *ReadFiles) ObservationStructured(input any, _ string, err error) aitypes.ToolObservation {
	request, ok := input.(*readFilesInputT)
	if !ok || request == nil {
		return aitypes.ToolObservation{Tool: t.Name(), Status: "error", Error: fmt.Sprintf("invalid structured input %T", input)}
	}
	files := make([]string, 0, len(request.Files))
	for _, file := range request.Files {
		files = append(files, file.Path)
	}
	observation := t.observation(files, err)
	if err == nil {
		for _, file := range request.Files {
			if file.StartLine != nil || file.EndLine != nil {
				encoded, _ := json.Marshal(file)
				observation.Inputs = append(observation.Inputs, string(encoded))
				observation.Counts["rangedFiles"]++
			}
		}
	}
	return observation
}

func (t *ReadFiles) observation(files []string, err error) aitypes.ToolObservation {
	observation := aitypes.ToolObservation{Tool: t.Name(), Status: "ok"}
	if err != nil {
		observation.Status = "error"
		observation.Error = err.Error()
		return observation
	}
	observation.FilesRead = files
	observation.Counts = map[string]int{"files": len(files)}
	return observation
}
