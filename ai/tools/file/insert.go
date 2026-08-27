package filetools

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/lmorg/ttyphoon/ai/agent"
	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
	"github.com/lmorg/ttyphoon/debug"
	"github.com/lmorg/ttyphoon/types"
)

type InsertLines struct {
	agent   aitypes.Agent
	enabled bool
}

func init() {
	agent.ToolsAdd(&InsertLines{})
}

//go:embed insert_description.md
var insertLinesDescription string

func (t *InsertLines) New(agent aitypes.Agent) (aitypes.Tool, error) {
	return &InsertLines{agent: agent, enabled: false}, nil
}

func (t *InsertLines) Enabled() bool { return t.enabled }
func (t *InsertLines) Toggle()       { t.enabled = !t.enabled }

func (t *InsertLines) Name() string        { return "insertLines" }
func (t *InsertLines) Path() string        { return "internal" }
func (t *InsertLines) Description() string { return insertLinesDescription }

type insertT struct {
	Line int    `json:"line"`
	Text string `json:"text"`
}

type insertLinesInputT struct {
	File    string    `json:"file"`
	Inserts []insertT `json:"inserts"`
}

func (t *InsertLines) Call(ctx context.Context, input string) (string, error) {
	debug.Log(input)

	var request insertLinesInputT
	if err := json.Unmarshal([]byte(input), &request); err != nil {
		return fmt.Sprintf("ERROR: input must be valid JSON matching the tool schema: %s", err), nil
	}

	if strings.TrimSpace(request.File) == "" {
		return "ERROR: 'file' is required", nil
	}
	if len(request.Inserts) == 0 {
		return "ERROR: 'inserts' must contain at least one insert", nil
	}

	filename, err := resolveWorkspacePath(t.agent.GetMeta().Pwd, request.File)
	if err != nil {
		return t.fail(fmt.Sprintf("ERROR '%s': %s", request.File, err)), nil
	}

	t.agent.Renderer().DisplayNotification(types.NOTIFY_INFO, t.agent.ServiceName()+" inserting lines into file: "+request.File)

	info, err := os.Stat(filename)
	if err != nil {
		return t.fail(fmt.Sprintf("ERROR '%s': cannot open file, use writeFile to create new files: %s", request.File, err)), nil
	}
	if info.IsDir() {
		return t.fail(fmt.Sprintf("ERROR '%s': path is a directory", request.File)), nil
	}

	b, err := os.ReadFile(filename)
	if err != nil {
		return t.fail(fmt.Sprintf("ERROR '%s': %s", request.File, err)), nil
	}

	updated, err := applyLineInserts(string(b), request.Inserts)
	if err != nil {
		return t.fail(fmt.Sprintf("ERROR '%s': %s", request.File, err)), nil
	}

	if err := writeFileAtomic(filename, []byte(updated), info.Mode().Perm()); err != nil {
		return t.fail(fmt.Sprintf("ERROR '%s': %s", request.File, err)), nil
	}

	result := fmt.Sprintf("INFO '%s': %d insert(s) applied successfully\n", request.File, len(request.Inserts))
	debug.Log(result)
	return result, nil
}

func (t *InsertLines) fail(message string) string {
	t.agent.Renderer().DisplayNotification(types.NOTIFY_ERROR, message)
	debug.Log(message)
	return message + "\nNo changes were written to disk.\n"
}

// applyLineInserts applies every insert or none. Line numbers all address the
// original content, so inserts are applied bottom-up to keep them valid.
func applyLineInserts(content string, inserts []insertT) (string, error) {
	lines, trailingNewline := splitLines(content)

	order := make([]int, len(inserts))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool {
		if inserts[order[a]].Line != inserts[order[b]].Line {
			return inserts[order[a]].Line > inserts[order[b]].Line
		}
		return order[a] > order[b]
	})

	for _, i := range order {
		if inserts[i].Line < 0 || inserts[i].Line > len(lines) {
			return "", fmt.Errorf("insert %d: line %d is out of range, the file has %d lines", i+1, inserts[i].Line, len(lines))
		}

		newLines, _ := splitLines(inserts[i].Text)
		if len(newLines) == 0 {
			continue
		}

		expanded := make([]string, 0, len(lines)+len(newLines))
		expanded = append(expanded, lines[:inserts[i].Line]...)
		expanded = append(expanded, newLines...)
		expanded = append(expanded, lines[inserts[i].Line:]...)
		lines = expanded
	}

	updated := strings.Join(lines, "\n")
	if trailingNewline && updated != "" {
		updated += "\n"
	}
	return updated, nil
}

// splitLines reports whether content ended with a newline so the terminator can
// be restored without gaining or losing a blank final line.
func splitLines(content string) (lines []string, trailingNewline bool) {
	if content == "" {
		return nil, false
	}

	trailingNewline = strings.HasSuffix(content, "\n")
	if trailingNewline {
		content = content[:len(content)-1]
	}

	return strings.Split(content, "\n"), trailingNewline
}
