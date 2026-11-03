package filetools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eino-contrib/jsonschema"
	"golang.org/x/tools/txtar"
)

func TestReadFilesSchemaPlacesBoundsOnIndividualFiles(t *testing.T) {
	reflector := jsonschema.Reflector{ExpandedStruct: true, DoNotReference: true, Anonymous: true}
	encoded, err := json.Marshal(reflector.ReflectFromType((&ReadFiles{}).InputType()))
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		Properties map[string]json.RawMessage `json:"properties"`
	}
	if err := json.Unmarshal(encoded, &schema); err != nil {
		t.Fatal(err)
	}
	if len(schema.Properties) != 1 || schema.Properties["files"] == nil {
		t.Fatalf("schema still exposes global bounds: %s", encoded)
	}
	var files struct {
		Items struct {
			Type       string                     `json:"type"`
			Properties map[string]json.RawMessage `json:"properties"`
			Required   []string                   `json:"required"`
		} `json:"items"`
	}
	if err := json.Unmarshal(schema.Properties["files"], &files); err != nil {
		t.Fatal(err)
	}
	if files.Items.Type != "object" || files.Items.Properties["path"] == nil || files.Items.Properties["startLine"] == nil || files.Items.Properties["endLine"] == nil || len(files.Items.Required) != 1 || files.Items.Required[0] != "path" {
		t.Fatalf("schema does not advertise independent optional ranges: %s", encoded)
	}
}

func TestReadFilesRejectsGlobalBoundsAndNullEntries(t *testing.T) {
	for _, input := range []string{
		`{"files":["main.go"],"startLine":5}`, `{"files":[{"path":"main.go"}],"endLine":20}`, `{"files":[null]}`,
	} {
		var request readFilesInputT
		if err := json.Unmarshal([]byte(input), &request); err == nil {
			t.Fatalf("accepted ambiguous or invalid input: %s", input)
		}
	}
}

func TestReadFilesSupportsIndependentRangesAndWholeFilesInOneCall(t *testing.T) {
	root := t.TempDir()
	contents := "one\ntwo\nthree\nfour\n"
	for _, name := range []string{"first.txt", "second.txt", "whole.txt", "legacy.txt"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(contents), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	tool := &ReadFiles{agent: &directoryTestAgent{pathTestAgent: pathTestAgent{projectRoot: root}, renderer: directoryTestRenderer{}}}
	var request readFilesInputT
	if err := json.Unmarshal([]byte(`{"files":[{"path":"first.txt","startLine":2,"endLine":3},{"path":"second.txt","startLine":4,"endLine":4},{"path":"whole.txt"},"legacy.txt"]}`), &request); err != nil {
		t.Fatal(err)
	}
	output, err := tool.CallStructured(context.Background(), &request)
	if err != nil {
		t.Fatal(err)
	}
	archive := txtar.Parse([]byte(output))
	want := []struct{ name, content string }{
		{"first.txt", "two\nthree\n"}, {"second.txt", "four\n"},
		{"whole.txt", contents}, {"legacy.txt", contents},
	}
	if len(archive.Files) != len(want) {
		t.Fatalf("expected four independent file reads: %q", output)
	}
	for index, expected := range want {
		if archive.Files[index].Name != expected.name || string(archive.Files[index].Data) != expected.content {
			t.Fatalf("wrong range for %s: %q", expected.name, archive.Files[index].Data)
		}
	}
}

func TestReadFilesOptionalLineRanges(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "lines.txt"), []byte("one\ntwo\nthree\nfour\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	tool := &ReadFiles{agent: &directoryTestAgent{pathTestAgent: pathTestAgent{projectRoot: root}, renderer: directoryTestRenderer{}}}
	for _, test := range []struct {
		name, bounds, want string
		invalid            bool
	}{
		{name: "whole file", want: "one\ntwo\nthree\nfour\n"},
		{name: "inclusive range", bounds: `,"startLine":2,"endLine":3`, want: "two\nthree\n"},
		{name: "single line", bounds: `,"startLine":3,"endLine":3`, want: "three\n"},
		{name: "from start", bounds: `,"endLine":2`, want: "one\ntwo\n"},
		{name: "through EOF", bounds: `,"startLine":3`, want: "three\nfour\n"},
		{name: "end beyond EOF", bounds: `,"startLine":3,"endLine":20`, want: "three\nfour\n"},
		{name: "start beyond EOF", bounds: `,"startLine":20,"endLine":30`, want: ""},
		{name: "zero start", bounds: `,"startLine":0`, invalid: true},
		{name: "negative end", bounds: `,"endLine":-1`, invalid: true},
		{name: "reversed range", bounds: `,"startLine":4,"endLine":2`, invalid: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			var request readFilesInputT
			if err := json.Unmarshal([]byte(`{"files":[{"path":"lines.txt"`+test.bounds+`}]}`), &request); err != nil {
				t.Fatal(err)
			}
			output, err := tool.CallStructured(context.Background(), &request)
			if test.invalid {
				if err == nil {
					t.Fatal("invalid line range was accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			archive := txtar.Parse([]byte(output))
			if len(archive.Files) != 1 || archive.Files[0].Name != "lines.txt" || string(archive.Files[0].Data) != test.want {
				t.Fatalf("read output = %q, want file contents %q", output, test.want)
			}
		})
	}
}

type countingLineReader struct {
	*strings.Reader
	readBytes int
}

func (reader *countingLineReader) Read(buffer []byte) (int, error) {
	count, err := reader.Reader.Read(buffer)
	reader.readBytes += count
	return count, err
}

func TestReadLineRangeStreamsAndPreservesLongLinesAndLineEndings(t *testing.T) {
	longLine := strings.Repeat("x", 128*1024)
	for _, test := range []struct {
		name, contents, want string
	}{
		{"CRLF", "one\r\ntwo\r\nthree\r\n", "two\r\n"},
		{"unterminated", "one\ntwo", "two"},
		{"long skipped line", longLine + "\ntwo\nthree\n", "two\n"},
		{"long selected line", "one\n" + longLine + "\nthree\n", longLine + "\n"},
		{"empty file", "", ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			data, err := readLineRange(context.Background(), strings.NewReader(test.contents), 2, 2)
			if err != nil || string(data) != test.want {
				t.Fatalf("ranged read mismatch: bytes=%d, want=%d, error=%v", len(data), len(test.want), err)
			}
		})
	}
	contents := "one\ntwo\n" + strings.Repeat("unrequested\n", 100_000)
	source := &countingLineReader{Reader: strings.NewReader(contents)}
	data, err := readLineRange(context.Background(), source, 1, 2)
	if err != nil || string(data) != "one\ntwo\n" || source.readBytes >= len(contents) {
		t.Fatalf("bounded read consumed the whole file: bytes=%d, error=%v", source.readBytes, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := readLineRange(ctx, strings.NewReader(contents), 1, 2); err != context.Canceled {
		t.Fatalf("cancelled read returned %v", err)
	}
}

func TestReadFilesRangeIsVisibleInApprovalAndObservation(t *testing.T) {
	start, end := 5, 20
	request := &readFilesInputT{Files: []readFileInputT{{Path: "main.go", StartLine: &start, EndLine: &end}, {Path: "README.md"}}}
	tool := &ReadFiles{}
	description := tool.CallDescription(context.Background(), request)
	if !strings.Contains(description, "5") || !strings.Contains(description, "20") || !strings.Contains(description, "inclusive") {
		t.Fatalf("approval omits the requested range: %q", description)
	}
	observation := tool.ObservationStructured(request, "", nil)
	if observation.Counts["rangedFiles"] != 1 || len(observation.Inputs) != 1 {
		t.Fatalf("observation omits range: %+v", observation)
	}
	var rangedFile readFileInputT
	if err := json.Unmarshal([]byte(observation.Inputs[0]), &rangedFile); err != nil || rangedFile.Path != "main.go" || *rangedFile.StartLine != 5 || *rangedFile.EndLine != 20 {
		t.Fatalf("observation lost the file/range association: %+v, %v", rangedFile, err)
	}
}
