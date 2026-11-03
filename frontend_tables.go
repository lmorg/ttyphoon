package main

import (
	"encoding/base64"
	"log"

	"github.com/lmorg/ttyphoon/utils/graphs"
	"github.com/lmorg/ttyphoon/utils/notes"
	"github.com/lmorg/ttyphoon/window/elements/element_table/tablecore"
)

// Tables below this many cells are rebuilt per query rather than retained, so
// the common case leaves nothing behind on the Go side at all.
const notesTableEphemeralCells = 2000

// NotesTableSeedT carries the evaluated cell values a table is built from.
// Formulas are already resolved by the frontend, so sorting acts on what the
// user can actually see.
type NotesTableSeedT struct {
	Headings []string   `json:"headings"`
	Rows     [][]string `json:"rows"`
}

// NotesTableRequestT is the frontend's view state plus, optionally, the data to
// rebuild the table from.
//
// Rendered state travels with every request; persisted state is restored when a
// table is first rendered so evicted tables remain recoverable.
type NotesTableRequestT struct {
	Key        tablecore.Key    `json:"key"`
	Workspace  string           `json:"workspace"`
	StateIndex int              `json:"stateIndex"`
	Seed       *NotesTableSeedT `json:"seed"`
	SortColumn int              `json:"sortColumn"`
	SortDesc   bool             `json:"sortDesc"`
	Filter     string           `json:"filter"`
}

// NotesTableResultT is the new view state and the source row indices to render.
type NotesTableResultT struct {
	// Missing asks the caller to resend the seed and retry once.
	Missing    bool                  `json:"missing"`
	Order      []int                 `json:"order"`
	SortColumn int                   `json:"sortColumn"`
	SortDesc   bool                  `json:"sortDesc"`
	Filter     string                `json:"filter"`
	Graph      *NotesTableGraphSpecT `json:"graph"`
	Error      string                `json:"error"`
}

type NotesTableGraphSpecT struct {
	Type              string   `json:"type"`
	Format            string   `json:"format"`
	Title             string   `json:"title"`
	CategoryColumn    int      `json:"categoryColumn"`
	ValueColumns      []int    `json:"valueColumns"`
	SeriesNames       []string `json:"seriesNames"`
	RowIndices        []int    `json:"rowIndices"`
	CategoryAxisTitle string   `json:"categoryAxisTitle"`
	ValueAxisTitle    string   `json:"valueAxisTitle"`
}

type NotesTableGraphResultT struct {
	SVG        string `json:"svg"`
	PNGDataURL string `json:"pngDataURL"`
	Error      string `json:"error"`
}

// NotesTableSort applies the terminal's heading-click rule to column (1-based).
func (a *WApp) NotesTableSort(req NotesTableRequestT, column int) NotesTableResultT {
	return a.notesTableApply(req, func(table *tablecore.Table) error {
		table.ToggleSort(column)
		return nil
	})
}

// NotesTableClearSort restores source order.
func (a *WApp) NotesTableClearSort(req NotesTableRequestT) NotesTableResultT {
	return a.notesTableApply(req, func(table *tablecore.Table) error {
		table.ClearSort()
		return nil
	})
}

// NotesTableFilter applies a raw SQL WHERE fragment; empty clears it.
func (a *WApp) NotesTableFilter(req NotesTableRequestT, where string) NotesTableResultT {
	return a.notesTableApply(req, func(table *tablecore.Table) error {
		return table.SetFilter(where)
	})
}

// NotesTableReconcile drops any retained table for this surface and document
// that the frontend no longer lists, which is how a missed cleanup self-heals.
func (a *WApp) NotesTableReconcile(surface, document string, keep []int) {
	a.notesTables.Reconcile(surface, document, keep)
}

// NotesTableDisposeAll releases every retained table.
func (a *WApp) NotesTableDisposeAll() {
	a.notesTables.Clear()
}

func (a *WApp) notesTableApply(req NotesTableRequestT, apply func(*tablecore.Table) error) NotesTableResultT {
	table, release, result := a.notesTableResolve(req)
	if table == nil {
		return result
	}
	if release != nil {
		defer release()
	}

	// Restore the caller's view state so retained and rebuilt tables behave
	// identically, then let the engine apply its own transition rules.
	table.SetSort(req.SortColumn, req.SortDesc)
	if err := table.SetFilter(req.Filter); err != nil {
		// A filter the frontend is already displaying should still load; if it
		// cannot, fall back to showing everything rather than failing the call.
		log.Printf("notes table: discarding unusable filter %q: %v", req.Filter, err)
	}

	if err := apply(table); err != nil {
		return NotesTableResultT{
			SortColumn: req.SortColumn,
			SortDesc:   req.SortDesc,
			Filter:     req.Filter,
			Error:      err.Error(),
		}
	}

	order, err := table.Order(0, 0)
	if err != nil {
		return NotesTableResultT{
			SortColumn: req.SortColumn,
			SortDesc:   req.SortDesc,
			Filter:     req.Filter,
			Error:      err.Error(),
		}
	}

	column, desc := table.SortState()
	state := notes.GetTableStateWithLegacyIndex(req.Workspace, req.Key.Document, req.StateIndex, req.Key.Index, table.Headings())
	result = NotesTableResultT{
		Order:      order,
		SortColumn: column,
		SortDesc:   desc,
		Filter:     table.Filter(),
		Graph:      graphSpecFromStored(state.Graph),
	}
	state.SortColumn = result.SortColumn
	state.SortDesc = result.SortDesc
	state.Filter = result.Filter
	notes.SetTableState(req.Workspace, req.Key.Document, req.StateIndex, table.Headings(), state)
	return result
}

// NotesTableRestore returns the persisted state for a newly rendered table.
func (a *WApp) NotesTableRestore(req NotesTableRequestT) NotesTableResultT {
	table, release, result := a.notesTableResolve(req)
	if table == nil {
		return result
	}
	if release != nil {
		defer release()
	}
	persisted := notes.GetTableStateWithLegacyIndex(req.Workspace, req.Key.Document, req.StateIndex, req.Key.Index, table.Headings())
	if err := table.SetFilter(persisted.Filter); err != nil {
		return NotesTableResultT{Error: err.Error()}
	}
	table.SetSort(persisted.SortColumn, persisted.SortDesc)
	order, err := table.Order(0, 0)
	if err != nil {
		return NotesTableResultT{Error: err.Error()}
	}
	column, desc := table.SortState()
	return NotesTableResultT{Order: order, SortColumn: column, SortDesc: desc, Filter: table.Filter(), Graph: graphSpecFromStored(persisted.Graph)}
}

// NotesTableSetGraph stores a graph specification using the table's SQL-state cache key.
func (a *WApp) NotesTableSetGraph(req NotesTableRequestT, graph *NotesTableGraphSpecT) NotesTableResultT {
	table, release, result := a.notesTableResolve(req)
	if table == nil {
		return result
	}
	if release != nil {
		defer release()
	}
	state := notes.GetTableStateWithLegacyIndex(req.Workspace, req.Key.Document, req.StateIndex, req.Key.Index, table.Headings())
	state.Graph = graphSpecToStored(graph)
	notes.SetTableState(req.Workspace, req.Key.Document, req.StateIndex, table.Headings(), state)
	return NotesTableResultT{Graph: graph}
}

// NotesTableRenderGraph renders the rows supplied by the caller as SVG or PNG.
func (a *WApp) NotesTableRenderGraph(req NotesTableRequestT, graph *NotesTableGraphSpecT) NotesTableGraphResultT {
	if req.Seed == nil {
		return NotesTableGraphResultT{Error: "table data was not supplied"}
	}
	if graph == nil {
		return NotesTableGraphResultT{Error: "graph specification was not supplied"}
	}
	if len(req.Seed.Headings) == 0 {
		return NotesTableGraphResultT{Error: "table headings were not supplied"}
	}
	table, release, err := a.notesTables.Build(req.Key, req.Seed.Headings, req.Seed.Rows)
	if err != nil {
		return NotesTableGraphResultT{Error: err.Error()}
	}
	defer release()
	if err := table.SetFilter(req.Filter); err != nil {
		return NotesTableGraphResultT{Error: err.Error()}
	}
	order, err := table.Order(0, 0)
	if err != nil {
		return NotesTableGraphResultT{Error: err.Error()}
	}
	selected := make(map[int]struct{}, len(graph.RowIndices))
	for _, index := range graph.RowIndices {
		selected[index] = struct{}{}
	}
	rows := make([][]string, 0, len(order))
	for _, sourceIndex := range order {
		if len(selected) > 0 {
			if _, ok := selected[sourceIndex]; !ok {
				continue
			}
		}
		if sourceIndex >= 0 && sourceIndex < len(req.Seed.Rows) {
			rows = append(rows, req.Seed.Rows[sourceIndex])
		}
	}
	if len(rows) == 0 {
		return NotesTableGraphResultT{Error: "no rows are selected for this graph"}
	}
	spec := graph.toGraphsSpec()
	spec.Format = graphs.SVG
	if graph.Format == string(graphs.PNG) {
		spec.Format = graphs.PNG
	}
	spec.RowIndices = nil
	data, err := graphs.RenderTable(graphs.Table{Headings: req.Seed.Headings, Rows: rows}, spec)
	if err != nil {
		return NotesTableGraphResultT{Error: err.Error()}
	}
	if spec.Format == graphs.PNG {
		return NotesTableGraphResultT{PNGDataURL: "data:image/png;base64," + base64.StdEncoding.EncodeToString(data)}
	}
	return NotesTableGraphResultT{SVG: string(data)}
}

func graphSpecFromStored(spec *graphs.Spec) *NotesTableGraphSpecT {
	if spec == nil {
		return nil
	}
	return &NotesTableGraphSpecT{
		Type:              string(spec.Type),
		Format:            string(spec.Format),
		Title:             spec.Title,
		CategoryColumn:    spec.CategoryColumn,
		ValueColumns:      append([]int(nil), spec.ValueColumns...),
		SeriesNames:       append([]string(nil), spec.SeriesNames...),
		RowIndices:        append([]int(nil), spec.RowIndices...),
		CategoryAxisTitle: spec.CategoryAxisTitle,
		ValueAxisTitle:    spec.ValueAxisTitle,
	}
}

func graphSpecToStored(spec *NotesTableGraphSpecT) *graphs.Spec {
	if spec == nil {
		return nil
	}
	converted := spec.toGraphsSpec()
	return &converted
}

func (spec *NotesTableGraphSpecT) toGraphsSpec() graphs.Spec {
	if spec == nil {
		return graphs.Spec{}
	}
	return graphs.Spec{
		Type:              graphs.Type(spec.Type),
		Format:            graphs.Format(spec.Format),
		Title:             spec.Title,
		CategoryColumn:    spec.CategoryColumn,
		ValueColumns:      append([]int(nil), spec.ValueColumns...),
		SeriesNames:       append([]string(nil), spec.SeriesNames...),
		RowIndices:        append([]int(nil), spec.RowIndices...),
		CategoryAxisTitle: spec.CategoryAxisTitle,
		ValueAxisTitle:    spec.ValueAxisTitle,
	}
}

// notesTableResolve returns the table for a request, building it from the seed
// when it is not retained. The returned release function is non-nil only for
// tables that were built ephemerally.
func (a *WApp) notesTableResolve(req NotesTableRequestT) (*tablecore.Table, func(), NotesTableResultT) {
	if table, ok := a.notesTables.Get(req.Key); ok {
		return table, nil, NotesTableResultT{}
	}

	if req.Seed == nil || len(req.Seed.Headings) == 0 {
		return nil, nil, NotesTableResultT{
			Missing:    true,
			SortColumn: req.SortColumn,
			SortDesc:   req.SortDesc,
			Filter:     req.Filter,
		}
	}

	if notesTableCells(req.Seed) <= notesTableEphemeralCells {
		table, release, err := a.notesTables.Build(req.Key, req.Seed.Headings, req.Seed.Rows)
		if err != nil {
			return nil, nil, NotesTableResultT{Error: err.Error()}
		}
		return table, release, NotesTableResultT{}
	}

	table, err := a.notesTables.Put(req.Key, req.Seed.Headings, req.Seed.Rows)
	if err != nil {
		return nil, nil, NotesTableResultT{Error: err.Error()}
	}

	return table, nil, NotesTableResultT{}
}

func notesTableCells(seed *NotesTableSeedT) int {
	return len(seed.Headings) * (len(seed.Rows) + 1)
}
