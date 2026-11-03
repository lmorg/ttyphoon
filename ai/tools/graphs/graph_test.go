package graphtools

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
)

func TestCreateGraphExposesStructuredInput(t *testing.T) {
	tool := &CreateGraph{}
	var structured aitypes.StructuredTool = tool
	if got := structured.InputType(); got != reflect.TypeOf(createGraphInputT{}) {
		t.Fatalf("InputType() = %v, want createGraphInputT", got)
	}
	if _, ok := structured.InputType().FieldByName("Format"); ok {
		t.Fatal("InputType() still exposes a format option")
	}
}

func TestCreateGraphDefaultsToPNGAndReturnsMarkdownImage(t *testing.T) {
	outputDir := t.TempDir()
	tool := &CreateGraph{}
	output, err := tool.create(nil, &createGraphInputT{
		Headings:       []string{"Month", "Sales"},
		Rows:           [][]string{{"Jan", "12"}, {"Feb", "18"}},
		Type:           "line",
		CategoryColumn: 0,
		ValueColumns:   []int{1},
		Title:          "Monthly sales",
	}, outputDir)
	if err != nil {
		t.Fatalf("create() error = %v", err)
	}

	path := graphPathFromOutput(output)
	if path == "" || filepath.Dir(path) != outputDir || filepath.Ext(path) != ".png" {
		t.Fatalf("output path = %q, want generated PNG in %q", path, outputDir)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", path, err)
	}
	if len(data) < 8 || string(data[:8]) != string([]byte{137, 80, 78, 71, 13, 10, 26, 10}) {
		t.Fatalf("file %q does not contain PNG data", path)
	}
	if !strings.Contains(output, "![Monthly sales]("+filepath.ToSlash(path)+")") {
		t.Fatalf("output %q does not contain the file-backed Markdown image", output)
	}

	observation := tool.ObservationStructured(&createGraphInputT{Title: "Monthly sales"}, output, nil)
	if observation.Status != "ok" || len(observation.Outputs) != 1 || observation.Outputs[0] != path {
		t.Fatalf("observation = %+v, want generated graph path", observation)
	}
}

func TestCreateGraphAlwaysWritesPNG(t *testing.T) {
	outputDir := t.TempDir()
	output, err := (&CreateGraph{}).create(nil, &createGraphInputT{
		Headings:       []string{"Month", "Sales"},
		Rows:           [][]string{{"Jan", "12"}, {"Feb", "18"}},
		Type:           "bar",
		CategoryColumn: 0,
		ValueColumns:   []int{1},
	}, outputDir)
	if err != nil {
		t.Fatalf("create() error = %v", err)
	}
	path := graphPathFromOutput(output)
	if filepath.Ext(path) != ".png" {
		t.Fatalf("output path = %q, want .png extension", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile(%q): %v", path, err)
	}
	if len(data) < 8 || string(data[:8]) != string([]byte{137, 80, 78, 71, 13, 10, 26, 10}) {
		t.Fatalf("file %q does not contain PNG data", path)
	}
}

func TestCreateGraphReturnsRendererErrorsToAgent(t *testing.T) {
	_, err := (&CreateGraph{}).create(nil, &createGraphInputT{
		Headings:       []string{"Category", "Value"},
		Rows:           [][]string{{"A", "1"}},
		Type:           "pie",
		CategoryColumn: 0,
		ValueColumns:   []int{1, 1},
	}, t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "exactly one value column") {
		t.Fatalf("create() error = %v, want pie mapping error", err)
	}
}
