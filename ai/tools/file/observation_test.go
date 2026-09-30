package filetools

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/lmorg/ttyphoon/ai/agent/aitypes"
	"github.com/lmorg/ttyphoon/utils/grep"
)

func TestFileToolsExposeDocumentedStructuredInputTypes(t *testing.T) {
	tests := []struct {
		name   string
		tool   aitypes.StructuredTool
		kind   reflect.Kind
		fields []string
	}{
		{name: "readFiles", tool: &ReadFiles{}, kind: reflect.Struct, fields: []string{"Files"}},
		{name: "readDirectory", tool: &Directory{}, kind: reflect.Struct, fields: []string{"Filter"}},
		{name: "writeFile", tool: &Write{}, kind: reflect.Struct, fields: []string{"Archive"}},
		{name: "insertLines", tool: &InsertLines{}, kind: reflect.Struct, fields: []string{"File", "Inserts"}},
		{name: "patchFile", tool: &PatchFile{}, kind: reflect.Struct, fields: []string{"File", "Edits"}},
		{name: "grep", tool: &Grep{}, kind: reflect.Struct, fields: []string{"Query", "Options", "Page"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			inputType := test.tool.InputType()
			if inputType.Kind() != test.kind {
				t.Fatalf("InputType() kind = %s, want %s", inputType.Kind(), test.kind)
			}
			for _, field := range test.fields {
				if _, ok := inputType.FieldByName(field); !ok {
					t.Errorf("InputType() %s has no %q field", inputType, field)
				}
			}
			if _, ok := test.tool.(aitypes.StructuredToolObservationProvider); !ok {
				t.Errorf("%s has no structured observation provider", test.name)
			}
		})
	}
}

func TestReadFilesObservationRecordsFilesRead(t *testing.T) {
	observation := (&ReadFiles{}).Observation(`["a.go","b.go"]`, "", nil)
	if observation.Tool != "readFiles" || observation.Status != "ok" {
		t.Fatalf("observation = %+v, want readFiles ok", observation)
	}
	if strings.Join(observation.FilesRead, ",") != "a.go,b.go" {
		t.Fatalf("FilesRead = %#v", observation.FilesRead)
	}
	if observation.Counts["files"] != 2 {
		t.Fatalf("file count = %d, want 2", observation.Counts["files"])
	}
}

func TestGrepObservationRecordsSearchSummary(t *testing.T) {
	result := grepReturnT{Results: []*grep.Result{
		{Path: "/tmp/a.go"},
		{Path: "/tmp/a.go"},
		{Path: "/tmp/b.go"},
	}}
	b, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	observation := (&Grep{}).Observation(`{"query":"needle","options":{"fileFilter":"*.go"}}`, string(b), nil)
	if len(observation.SearchesRun) != 1 {
		t.Fatalf("SearchesRun = %d, want 1", len(observation.SearchesRun))
	}
	search := observation.SearchesRun[0]
	if search.Query != "needle" || search.FileFilter != "*.go" || search.ResultCount != 3 {
		t.Fatalf("search observation = %+v", search)
	}
	if strings.Join(search.TopPaths, ",") != "/tmp/a.go,/tmp/b.go" {
		t.Fatalf("TopPaths = %#v", search.TopPaths)
	}
}

func TestGrepStructuredToolInputAndObservation(t *testing.T) {
	tool := &Grep{}
	if got := tool.InputType(); got != reflect.TypeOf(grepInputT{}) {
		t.Fatalf("InputType() = %v, want grepInputT", got)
	}

	output, err := tool.CallStructured(context.Background(), &grepInputT{Page: 0})
	if err != nil {
		t.Fatalf("CallStructured: %v", err)
	}
	var result grepReturnT
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		t.Fatalf("CallStructured output is not JSON: %v", err)
	}
	if result.Error != "Page numbers cannot be 0 nor negative" {
		t.Fatalf("structured page result = %+v", result)
	}

	observation := tool.ObservationStructured(&grepInputT{
		Query:   "needle",
		Options: grepOptionsInputT{FileFilter: "*.go"},
	}, `{"results":[{"path":"/tmp/a.go"}],"pageNumber":1}`, nil)
	if len(observation.SearchesRun) != 1 || observation.SearchesRun[0].Query != "needle" || observation.SearchesRun[0].FileFilter != "*.go" {
		t.Fatalf("structured observation = %+v", observation)
	}
}

func TestDirectoryObservationRecordsListedRootAndCount(t *testing.T) {
	agent := &pathTestAgent{projectRoot: "/tmp/project"}
	observation := (&Directory{agent: agent}).Observation("*.go", `[{"Name":"main.go","IsDir":false}]`, nil)
	if observation.Tool != "readDirectory" || observation.Status != "ok" {
		t.Fatalf("observation = %+v, want readDirectory ok", observation)
	}
	if strings.Join(observation.DirectoriesListed, ",") != "/tmp/project" {
		t.Fatalf("DirectoriesListed = %#v", observation.DirectoriesListed)
	}
	if observation.Counts["matches"] != 1 {
		t.Fatalf("matches = %d, want 1", observation.Counts["matches"])
	}
}

func TestWriteObservationRecordsFilesModified(t *testing.T) {
	input := "-- a.go --\npackage main\n-- b.go --\npackage main\n"
	observation := (&Write{}).Observation(input, "", nil)
	if observation.Tool != "writeFile" || observation.Status != "ok" {
		t.Fatalf("observation = %+v, want writeFile ok", observation)
	}
	if strings.Join(observation.FilesModified, ",") != "a.go,b.go" {
		t.Fatalf("FilesModified = %#v", observation.FilesModified)
	}
}

func TestPatchAndInsertObservationRecordFailures(t *testing.T) {
	patch := (&PatchFile{}).Observation(`{"file":"a.go","edits":[{"old":"x","new":"y"}]}`, "ERROR 'a.go': no match", nil)
	if patch.Status != "error" || strings.Join(patch.FilesModified, ",") != "a.go" || patch.Counts["edits"] != 1 {
		t.Fatalf("patch observation = %+v", patch)
	}

	insert := (&InsertLines{}).Observation(`{"file":"b.go","inserts":[{"line":1,"text":"x"},{"line":2,"text":"y"}]}`, "", errors.New("boom"))
	if insert.Status != "error" || strings.Join(insert.FilesModified, ",") != "b.go" || insert.Counts["inserts"] != 2 || insert.Error != "boom" {
		t.Fatalf("insert observation = %+v", insert)
	}
}
