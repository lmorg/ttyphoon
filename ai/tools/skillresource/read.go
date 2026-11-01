package skillresource

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"unicode/utf8"

	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
)

const maxSkillResourceBytes = 1 << 20

type ReadSkillResource struct {
	agent   aitypes.Agent
	enabled bool
}

func init() {
	agent.ToolsAdd(&ReadSkillResource{})
}

//go:embed description.md
var description string

func (t *ReadSkillResource) New(agt aitypes.Agent) (aitypes.Tool, error) {
	return &ReadSkillResource{agent: agt, enabled: true}, nil
}

func (t *ReadSkillResource) Enabled() bool {
	if !t.enabled || t.agent == nil {
		return false
	}
	meta := t.agent.GetMeta()
	return meta != nil && len(meta.SkillResources) > 0
}

func (t *ReadSkillResource) Toggle()             { t.enabled = !t.enabled }
func (t *ReadSkillResource) Name() string        { return "readSkillResource" }
func (t *ReadSkillResource) Path() string        { return "internal" }
func (t *ReadSkillResource) Description() string { return strings.TrimSpace(description) }
func (t *ReadSkillResource) DefaultPermissions() aitypes.DefaultPermissions {
	return aitypes.DefaultPermissions{Invocation: "alwaysAllow", Subagents: "allow"}
}

type readSkillResourceInput struct {
	Skill     string `json:"skill"`
	Path      string `json:"path,omitempty"`
	StartLine *int   `json:"startLine,omitempty"`
	EndLine   *int   `json:"endLine,omitempty"`
}

func (t *ReadSkillResource) InputType() reflect.Type { return reflect.TypeOf(readSkillResourceInput{}) }

func (t *ReadSkillResource) CallStructured(ctx context.Context, input any) (string, error) {
	request, ok := input.(*readSkillResourceInput)
	if !ok || request == nil {
		return "", fmt.Errorf("readSkillResource received an invalid structured input %T", input)
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return readSkillResource(agent.SkillResourceDirectoriesFromContext(ctx), request.Skill, request.Path, request.StartLine, request.EndLine)
}

func (t *ReadSkillResource) CallDescription(_ context.Context, input any) string {
	request, ok := input.(*readSkillResourceInput)
	if !ok || request == nil {
		return "Read a file or list a directory in an active skill"
	}
	return fmt.Sprintf("Read skill %q resource %q", request.Skill, request.Path)
}

func (t *ReadSkillResource) ObservationStructured(input any, _ string, err error) aitypes.ToolObservation {
	request, ok := input.(*readSkillResourceInput)
	if !ok || request == nil {
		return aitypes.ToolObservation{Tool: t.Name(), Status: "error", Error: fmt.Sprintf("invalid input %T", input)}
	}
	observation := aitypes.ToolObservation{Tool: t.Name(), Status: "ok", FilesRead: []string{request.Skill + "/" + request.Path}}
	if err != nil {
		observation.Status = "error"
		observation.Error = err.Error()
	}
	return observation
}

func readSkillResource(directories map[string]string, skillName, resourcePath string, startLine, endLine *int) (string, error) {
	root, ok := directories[strings.TrimSpace(skillName)]
	if !ok || root == "" {
		return "", fmt.Errorf("skill %q is not active for this request", skillName)
	}

	path := strings.TrimSpace(resourcePath)
	if path == "" {
		path = "."
	}
	if filepath.IsAbs(path) {
		return "", fmt.Errorf("skill resource path must be relative")
	}
	cleanPath := filepath.Clean(filepath.FromSlash(path))
	if cleanPath == ".." || strings.HasPrefix(cleanPath, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("skill resource path escapes its skill directory")
	}

	root, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", fmt.Errorf("resolve skill directory: %w", err)
	}
	resource := filepath.Join(root, cleanPath)
	resource, err = filepath.EvalSymlinks(resource)
	if err != nil {
		return "", fmt.Errorf("resolve skill resource: %w", err)
	}
	relative, err := filepath.Rel(root, resource)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("skill resource path escapes its skill directory")
	}

	info, err := os.Stat(resource)
	if err != nil {
		return "", fmt.Errorf("stat skill resource: %w", err)
	}
	if info.IsDir() {
		if startLine != nil || endLine != nil {
			return "", fmt.Errorf("line ranges can only be used with files")
		}
		return listSkillResources(skillName, relative, resource)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("skill resource is not a regular file")
	}
	if info.Size() > maxSkillResourceBytes {
		return "", fmt.Errorf("skill resource exceeds the %d-byte read limit", maxSkillResourceBytes)
	}
	data, err := os.ReadFile(resource)
	if err != nil {
		return "", fmt.Errorf("read skill resource: %w", err)
	}
	if bytes.IndexByte(data, 0) >= 0 || !utf8.Valid(data) {
		return "", fmt.Errorf("skill resource is not UTF-8 text")
	}
	data, err = selectLines(data, startLine, endLine)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("Skill resource: %s/%s\nAbsolute path: %s\n\n%s", skillName, filepath.ToSlash(relative), resource, string(data)), nil
}

func listSkillResources(skillName, relative, directory string) (string, error) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return "", fmt.Errorf("list skill resources: %w", err)
	}
	if relative == "." {
		relative = ""
	}
	var output strings.Builder
	fmt.Fprintf(&output, "Resources in skill %q directory %q:\n", skillName, filepath.ToSlash(relative))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() {
			name += "/"
		}
		fmt.Fprintf(&output, "- %s\n", filepath.ToSlash(filepath.Join(relative, name)))
	}
	return output.String(), nil
}

func selectLines(data []byte, startLine, endLine *int) ([]byte, error) {
	if startLine == nil && endLine == nil {
		return data, nil
	}
	start, end := 1, 0
	if startLine != nil {
		start = *startLine
	}
	if endLine != nil {
		end = *endLine
	}
	if start < 1 || (end != 0 && end < start) {
		return nil, fmt.Errorf("invalid skill resource line range")
	}
	lines := strings.Split(string(data), "\n")
	if start > len(lines) {
		return []byte{}, nil
	}
	if end == 0 || end > len(lines) {
		end = len(lines)
	}
	return []byte(strings.Join(lines[start-1:end], "\n")), nil
}
