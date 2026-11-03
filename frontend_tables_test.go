package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image/png"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lmorg/ttyphoon/utils/graphs"
	"github.com/lmorg/ttyphoon/utils/notes"
	"github.com/lmorg/ttyphoon/window/elements/element_table/tablecore"
)

func newTableTestApp() *WApp {
	return &WApp{notesTables: tablecore.NewRegistry()}
}

func testTableKey() tablecore.Key {
	return tablecore.Key{Surface: "csv", Document: "/tmp/doc.csv", Index: 0}
}

func smallSeed() *NotesTableSeedT {
	return &NotesTableSeedT{
		Headings: []string{"n"},
		Rows:     [][]string{{"30"}, {"10"}, {"20"}},
	}
}

func largeSeed() *NotesTableSeedT {
	rows := make([][]string, 0, 1500)
	for i := 1500; i > 0; i-- {
		rows = append(rows, []string{fmt.Sprintf("%d", i), "x"})
	}

	return &NotesTableSeedT{Headings: []string{"n", "s"}, Rows: rows}
}

func assertResultOrder(t *testing.T, result NotesTableResultT, want ...int) {
	t.Helper()

	if result.Error != "" {
		t.Fatalf("unexpected error: %s", result.Error)
	}
	if len(result.Order) != len(want) {
		t.Fatalf("order = %v, want %v", result.Order, want)
	}
	for i := range want {
		if result.Order[i] != want[i] {
			t.Fatalf("order = %v, want %v", result.Order, want)
		}
	}
}

// Without a seed and without a retained table there is nothing to query, so the
// frontend must be told to resend rather than shown a broken table.
func TestNotesTableSort_ReportsMissingWithoutSeed(t *testing.T) {
	app := newTableTestApp()

	result := app.NotesTableSort(NotesTableRequestT{Key: testTableKey()}, 1)

	if !result.Missing {
		t.Fatalf("Missing = false, want true when the table is unknown and no seed was sent")
	}
}

func TestNotesTableSort_SortsFromSeed(t *testing.T) {
	app := newTableTestApp()

	result := app.NotesTableSort(NotesTableRequestT{Key: testTableKey(), Seed: smallSeed()}, 1)

	assertResultOrder(t, result, 1, 2, 0)
	if result.SortColumn != 1 || result.SortDesc {
		t.Fatalf("state = (%d, %v), want (1, false)", result.SortColumn, result.SortDesc)
	}
}

// The toggle rule lives in Go, but the state it toggles from travels with the
// request, so a rebuilt table flips direction just like a retained one.
func TestNotesTableSort_TogglesUsingRequestState(t *testing.T) {
	app := newTableTestApp()

	result := app.NotesTableSort(NotesTableRequestT{
		Key:        testTableKey(),
		Seed:       smallSeed(),
		SortColumn: 1,
		SortDesc:   false,
	}, 1)

	if !result.SortDesc {
		t.Fatal("SortDesc = false, want the second click on the same column to descend")
	}
	assertResultOrder(t, result, 0, 2, 1)
}

func TestNotesTableClearSort_RestoresSourceOrder(t *testing.T) {
	app := newTableTestApp()

	result := app.NotesTableClearSort(NotesTableRequestT{
		Key:        testTableKey(),
		Seed:       smallSeed(),
		SortColumn: 1,
	})

	assertResultOrder(t, result, 0, 1, 2)
	if result.SortColumn != tablecore.Unsorted {
		t.Fatalf("SortColumn = %d, want Unsorted", result.SortColumn)
	}
}

func TestNotesTableFilter_RestrictsRows(t *testing.T) {
	app := newTableTestApp()

	result := app.NotesTableFilter(NotesTableRequestT{Key: testTableKey(), Seed: smallSeed()}, `"n" > 15`)

	assertResultOrder(t, result, 0, 2)
	if result.Filter != `"n" > 15` {
		t.Fatalf("Filter = %q, want it echoed back", result.Filter)
	}
}

// A bad filter must surface as an error while leaving the caller's state intact.
func TestNotesTableFilter_InvalidFilterReportsError(t *testing.T) {
	app := newTableTestApp()

	result := app.NotesTableFilter(NotesTableRequestT{Key: testTableKey(), Seed: smallSeed()}, "not valid sql")

	if result.Error == "" {
		t.Fatal("Error = empty, want the invalid filter reported")
	}
	if result.Missing {
		t.Fatal("Missing = true, want an invalid filter not to look like a lost table")
	}
}

// Small tables must not accumulate on the Go side at all.
func TestNotesTableSort_SmallTablesAreNotRetained(t *testing.T) {
	app := newTableTestApp()

	app.NotesTableSort(NotesTableRequestT{Key: testTableKey(), Seed: smallSeed()}, 1)

	if got := app.notesTables.Len(); got != 0 {
		t.Fatalf("retained tables = %d, want 0 for a small table", got)
	}
}

// Large tables are retained so the data need not be resent on every click.
func TestNotesTableSort_LargeTablesAreRetainedAndReusable(t *testing.T) {
	app := newTableTestApp()

	first := app.NotesTableSort(NotesTableRequestT{Key: testTableKey(), Seed: largeSeed()}, 1)
	if first.Error != "" {
		t.Fatalf("unexpected error: %s", first.Error)
	}
	if got := app.notesTables.Len(); got != 1 {
		t.Fatalf("retained tables = %d, want 1 for a large table", got)
	}

	// No seed this time: it must be answered from the retained table.
	second := app.NotesTableSort(NotesTableRequestT{
		Key:        testTableKey(),
		SortColumn: first.SortColumn,
		SortDesc:   first.SortDesc,
	}, 1)

	if second.Missing {
		t.Fatal("Missing = true, want the retained table to answer without a seed")
	}
	if !second.SortDesc {
		t.Fatal("SortDesc = false, want the retained table to toggle to descending")
	}
}

func TestNotesTableReconcile_DropsUndeclaredTables(t *testing.T) {
	app := newTableTestApp()

	app.NotesTableSort(NotesTableRequestT{Key: testTableKey(), Seed: largeSeed()}, 1)
	if app.notesTables.Len() != 1 {
		t.Fatalf("setup failed: retained = %d", app.notesTables.Len())
	}

	app.NotesTableReconcile("csv", "/tmp/doc.csv", nil)

	if got := app.notesTables.Len(); got != 0 {
		t.Fatalf("retained tables = %d, want 0 after reconcile dropped it", got)
	}
}

func TestNotesTableDisposeAll_ReleasesEverything(t *testing.T) {
	app := newTableTestApp()

	app.NotesTableSort(NotesTableRequestT{Key: testTableKey(), Seed: largeSeed()}, 1)
	app.NotesTableDisposeAll()

	if got := app.notesTables.Len(); got != 0 {
		t.Fatalf("retained tables = %d, want 0 after dispose", got)
	}
}

// After eviction the frontend resends the seed; the result must be identical to
// what the retained table would have produced.
func TestNotesTableSort_RecreateAfterEvictionMatchesRetained(t *testing.T) {
	app := newTableTestApp()

	retained := app.NotesTableSort(NotesTableRequestT{Key: testTableKey(), Seed: largeSeed()}, 1)
	app.NotesTableDisposeAll()

	missing := app.NotesTableSort(NotesTableRequestT{Key: testTableKey()}, 1)
	if !missing.Missing {
		t.Fatal("Missing = false, want the evicted table to request a reseed")
	}

	rebuilt := app.NotesTableSort(NotesTableRequestT{Key: testTableKey(), Seed: largeSeed()}, 1)

	if len(rebuilt.Order) != len(retained.Order) {
		t.Fatalf("rebuilt order length = %d, want %d", len(rebuilt.Order), len(retained.Order))
	}
	for i := range retained.Order {
		if rebuilt.Order[i] != retained.Order[i] {
			t.Fatalf("rebuilt order diverged at %d: %d vs %d", i, rebuilt.Order[i], retained.Order[i])
		}
	}
}

func TestNotesTableGraphStatePersistsByWorkspaceAndDocument(t *testing.T) {
	app := newTableTestApp()
	workspace := t.TempDir()
	viewRequest := NotesTableRequestT{
		Key:       tablecore.Key{Surface: "preview", Document: filepath.Join(workspace, "table.md"), Index: 0},
		Workspace: workspace,
		Seed: &NotesTableSeedT{
			Headings: []string{"Category", "Value"},
			Rows:     [][]string{{"A", "1"}, {"B", "2"}},
		},
	}
	spec := &NotesTableGraphSpecT{Type: string(graphs.Line), CategoryColumn: 0, ValueColumns: []int{1}, RowIndices: []int{1}}
	runRequest := viewRequest
	runRequest.Key.Surface = "jupyter"
	runRequest.Key.Index = 1
	if result := app.NotesTableSetGraph(runRequest, spec); result.Error != "" {
		t.Fatalf("NotesTableSetGraph() error = %q", result.Error)
	}
	app.NotesTableDisposeAll()

	restored := app.NotesTableRestore(viewRequest)
	if restored.Error != "" {
		t.Fatalf("NotesTableRestore() error = %q", restored.Error)
	}
	if restored.Graph == nil || restored.Graph.Type != spec.Type || restored.Graph.CategoryColumn != spec.CategoryColumn || len(restored.Graph.ValueColumns) != 1 || restored.Graph.ValueColumns[0] != 1 || len(restored.Graph.RowIndices) != 1 || restored.Graph.RowIndices[0] != 1 {
		t.Fatalf("restored graph = %+v, want %+v", restored.Graph, spec)
	}

	otherWorkspace := viewRequest
	otherWorkspace.Workspace = filepath.Join(workspace, "other")
	otherWorkspace.Key.Document = viewRequest.Key.Document
	other := app.NotesTableRestore(otherWorkspace)
	if other.Graph != nil {
		t.Fatalf("other workspace graph = %+v, want no graph state", other.Graph)
	}
}

func TestNotesTableRestoreMigratesGraphFromLegacySurfaceIndex(t *testing.T) {
	app := newTableTestApp()
	workspace := t.TempDir()
	document := filepath.Join(workspace, "legacy-run-table.md")
	headings := []string{"Category", "Value"}
	legacySpec := &graphs.Spec{Type: graphs.Bar, CategoryColumn: 0, ValueColumns: []int{1}}
	notes.SetTableState(workspace, document, 1, headings, notes.TableState{Graph: legacySpec})

	request := NotesTableRequestT{
		Key:       tablecore.Key{Surface: "preview", Document: document, Index: 0},
		Workspace: workspace,
		Seed: &NotesTableSeedT{
			Headings: headings,
			Rows:     [][]string{{"A", "1"}},
		},
	}
	result := app.NotesTableRestore(request)
	if result.Error != "" {
		t.Fatalf("NotesTableRestore() error = %q", result.Error)
	}
	if result.Graph == nil || result.Graph.Type != string(graphs.Bar) {
		t.Fatalf("restored graph = %+v, want legacy graph spec", result.Graph)
	}

	migrated := notes.GetTableState(workspace, document, 0, headings)
	if migrated.Graph == nil || migrated.Graph.Type != graphs.Bar {
		t.Fatalf("stable graph state = %+v, want migrated legacy graph", migrated.Graph)
	}
}

func TestNotesTableRenderGraphPNG(t *testing.T) {
	app := newTableTestApp()
	request := NotesTableRequestT{
		Key:    tablecore.Key{Surface: "preview", Document: filepath.Join(t.TempDir(), "graph.md"), Index: 0},
		Filter: `"Value" > 1`,
		Seed:   &NotesTableSeedT{Headings: []string{"Category", "Value"}, Rows: [][]string{{"Excluded", "1"}, {"Selected", "2"}}},
	}
	spec := &NotesTableGraphSpecT{Type: string(graphs.Bar), Format: string(graphs.PNG), CategoryColumn: 0, ValueColumns: []int{1}, RowIndices: []int{1}}
	result := app.NotesTableRenderGraph(request, spec)
	if result.Error != "" || result.SVG != "" || !strings.HasPrefix(result.PNGDataURL, "data:image/png;base64,") {
		t.Fatalf("PNG result error = %q, SVG = %q, data URL prefix valid = %v", result.Error, result.SVG, strings.HasPrefix(result.PNGDataURL, "data:image/png;base64,"))
	}
	data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(result.PNGDataURL, "data:image/png;base64,"))
	if err != nil {
		t.Fatal(err)
	}
	image, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	if image.Bounds().Dx() <= 0 || image.Bounds().Dy() <= 0 {
		t.Fatal("PNG graph has empty dimensions")
	}
}

func TestNotesTableRenderGraphIntersectsSelectedRowsWithSQLFilter(t *testing.T) {
	app := newTableTestApp()
	request := NotesTableRequestT{
		Key:       tablecore.Key{Surface: "preview", Document: filepath.Join(t.TempDir(), "graph-filter.md"), Index: 0},
		Workspace: t.TempDir(),
		Filter:    `"Value" > 1`,
		Seed: &NotesTableSeedT{
			Headings: []string{"Category", "Value"},
			Rows: [][]string{
				{"ExcludeBySQL", "1"},
				{"SelectedMatch", "2"},
				{"MatchButNotSelected", "3"},
			},
		},
	}
	result := app.NotesTableRenderGraph(request, &NotesTableGraphSpecT{
		Type:           string(graphs.Bar),
		CategoryColumn: 0,
		ValueColumns:   []int{1},
		RowIndices:     []int{0, 1},
	})
	if result.Error != "" {
		t.Fatalf("NotesTableRenderGraph() error = %q", result.Error)
	}
	if !strings.Contains(result.SVG, "SelectedMatch") {
		t.Fatalf("SVG does not contain selected SQL match: %s", result.SVG)
	}
	for _, excluded := range []string{"ExcludeBySQL", "MatchButNotSelected"} {
		if strings.Contains(result.SVG, excluded) {
			t.Fatalf("SVG contains excluded row %q", excluded)
		}
	}
}
