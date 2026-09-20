package filetools

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/lmorg/ttyphoon/utils/grep"
)

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
