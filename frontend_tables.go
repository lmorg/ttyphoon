package main

import (
	"log"

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
// View state travels with every request rather than living on the Go side, so a
// dropped or evicted table costs nothing but the data itself.
type NotesTableRequestT struct {
	Key        tablecore.Key    `json:"key"`
	Seed       *NotesTableSeedT `json:"seed"`
	SortColumn int              `json:"sortColumn"`
	SortDesc   bool             `json:"sortDesc"`
	Filter     string           `json:"filter"`
}

// NotesTableResultT is the new view state and the source row indices to render.
type NotesTableResultT struct {
	// Missing asks the caller to resend the seed and retry once.
	Missing    bool   `json:"missing"`
	Order      []int  `json:"order"`
	SortColumn int    `json:"sortColumn"`
	SortDesc   bool   `json:"sortDesc"`
	Filter     string `json:"filter"`
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
	return NotesTableResultT{
		Order:      order,
		SortColumn: column,
		SortDesc:   desc,
		Filter:     table.Filter(),
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
