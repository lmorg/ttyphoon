# 0028 — Reuse the Go table engine for Notes table sorting and filtering

## Status

**Accepted and implemented** — reviewed, open questions resolved (see Resolved
decisions).

## Context

Tables are rendered in two completely separate places, with two completely
separate implementations of sorting:

**Terminal (Go)** — `window/elements/element_table/`. Data is loaded into an
in-memory SQLite database and *all* sorting, filtering and pagination is done
in SQL. Sort state is two fields on the element (`orderByIndex`, `orderDesc`),
filter state is one field (`filter`) holding a raw SQL `WHERE` clause.

**Notes (JavaScript)** — `frontend/src/notes.js` now delegates sort and filter
state to `window/elements/element_table/tablecore` through Wails bindings.
The frontend only reorders existing rows and preserves the DOM needed for
formulas, editing, resizing and word wrap.

The two behave differently in almost every respect:

| | Terminal | Notes today |
|---|---|---|
| Trigger | Left-click heading toggles sort | Left-click heading opens a 6-item menu |
| Sort options | ASC/DESC, type inferred | Explicit "by number" / "by characters", asc/desc |
| Clear sort | Right-click heading | Menu item "Clear sorting" |
| Numeric sort | SQLite numeric affinity | `parseFloat(x) || 0` |
| Text sort | `ORDER BY lower(col)` | `localeCompare` |
| Filtering | Raw SQL `WHERE`, double-click a data row | None |
| Engine | SQLite | JS array sort |

Note the current Notes binding is a **left**-click (`th.addEventListener('click', …)`,
line 3758) that opens a context menu — it is not a right-click menu.

The JS implementation also has behavioural quirks the SQL engine does not:
`parseFloat(aText) || 0` coerces every non-numeric cell to `0`, so mixed
columns sort nonsensically; and sorting by "number" on a text column silently
produces an arbitrary order.

The replacement engine is wired into four surfaces: the CSV view, markdown
preview, Jupyter view and AI output panel.

### Requirements being captured

From the request, verbatim:

1. Strip *all* the JS logic around filtering and sorting of Notes tables.
2. Reuse the Go package for that instead.
3. Column resizing must still work.
4. Cell functions (formulas / table expressions) must still work.
5. Word wrap must still work.
6. Sorting must model the same behaviour as the terminal Go package,
   **including the same mouse button interactions**.
7. The SQL filter must work, but reached by **right-click → "SQL filter…"**
   rather than the terminal's double-click.

## Decision

### 1. Extract a headless table engine

`element_table` cannot be called from a Wails binding as-is: it is built around
`types.Renderer` and `types.Tile`, it draws into the terminal, and its sort
state is entangled with terminal scroll offsets and viewport height.

Extract the data/query core into a new package —
`window/elements/element_table/tablecore` —
containing everything that is already renderer-agnostic:

- table construction (`createDb`, `createTable`, `insertRecords` from `db.go`)
- numeric column detection (`strconv.ParseFloat` on the first data row,
  `element_table.go` line 157)
- sort state and the toggle rules
- filter state
- SQL generation (`sqlWhere`, `sqlString` from `db_tables.go`)
- row retrieval and row count

Sketch:

```go
package tablecore

type Table struct { /* db, name, headings, isNumber, filter, sortCol, sortDesc */
}

func New(headings []string, rows [][]string) (*Table, error)

func (t *Table) ToggleSort(column int) // terminal left-click rule
func (t *Table) ClearSort()            // terminal right-click rule
func (t *Table) SetFilter(where string) error
func (t *Table) Order(limit, offset int) ([]int, error) // source row indices
func (t *Table) Rows(limit, offset int) ([][]string, error)
func (t *Table) Count() (int, error)
func (t *Table) Close() error
```

`ToggleSort` encodes the terminal's exact rule (`mouse.go` lines 85–92):

```go
package main

func main() {
	if t.sortCol == column {
		t.sortDesc = !t.sortDesc
	} else {
		t.sortCol = column
		t.sortDesc = false
	}

}
```

`limit <= 0` means "all rows" — the terminal keeps passing its viewport height,
Notes passes `0`.

`element_table` is then refactored to delegate to this package, keeping only
rendering, scroll offsets and mouse dispatch. **This is the crux of the
proposal**: parity is guaranteed by construction because there is exactly one
implementation of the semantics, not two that have to be kept in step.

### 2. Return a row permutation, not cell content

The single most important design decision. Go must **not** return rendered
rows to the frontend.

Notes table cells are not inert text. They carry `data-formula` attributes,
`contenteditable` editing state, `.notes-cellref` labels, `.notes-table-col-resize-handle`
spans, and bound event listeners. Replacing cell content from Go would destroy
all of it and break requirements 3, 4 and 5.

Instead, Go returns an **ordered list of source row indices** — the permutation
produced by `ORDER BY` and `WHERE`. The frontend then:

- **sorts** by reordering the existing `<tr>` nodes into that order;
- **filters** by hiding `<tr>` nodes whose index is absent from the result.

No cell is ever rewritten, so formulas, editing, resize handles, cell
references and word wrap are untouched by construction.

This works because insertion order into SQLite is source order, so SQLite's
implicit `rowid` *is* the source row index. The engine selects it explicitly
rather than relying on `SELECT *`.

### 3. Sort the values the user can see

Notes evaluates formulas at render time; a cell showing `42` may hold
`=R[0]C[1]+5`. Sorting must act on the rendered value, so the frontend seeds
the Go table with **evaluated display values**, reusing the existing
`getTableCellTextContent()` helper (which already strips sort icons and cell
refs).

Consequence: the Go-side table is a snapshot. Any edit, formula recompute, or
row/column insert invalidates it and requires a rebuild.

Note also that A1/R1C1 cell references remain bound to *source* position.
Sorting is a view transformation only — it must never rewrite the underlying
markdown or CSV. This matches today's behaviour (`originalSortOrder` restores
the original order) and must be preserved.

### 4. Wails bindings and lifetime

Retaining a SQLite database per open table is the main *new* failure mode this
change introduces, so lifetime is designed to be self-healing rather than to
depend on the frontend remembering to free things.

Five decisions, in order of how much they contribute:

**(a) Deterministic keys, not opaque handles.** A table is addressed by a key
the frontend can always reconstruct from what it is already rendering:

```go
package main

func main() {
	type TableKeyT struct {
		Surface  string `json:"surface"`  // "csv" | "preview" | "jupyter" | "ai"
		Document string `json:"document"` // file path, "" for the AI panel
		Index    int    `json:"index"`    // nth table within that surface
	}

}
```

This follows the existing `lspDocs` registry, which is keyed by absolute path
rather than by a handed-out handle. Because the key is reproducible, losing
server-side state is always recoverable, a webview reload is harmless, and
nothing needs to be persisted across a restart.

**(b) Recreate-on-miss makes eviction safe.** Every query returns a `Missing`
flag rather than failing when the key is unknown:

```go
type TableResultT struct {
    Missing    bool  `json:"missing"`    // re-seed and retry once
    Order      []int `json:"order"`      // source row indices
    SortColumn int   `json:"sortColumn"`
    SortDesc   bool  `json:"sortDesc"`
    Filter     string `json:"filter"`
}

func (a *WApp) NotesTableSort(key TableKeyT, column int, seed *TableSeedT) TableResultT
func (a *WApp) NotesTableClearSort(key TableKeyT, seed *TableSeedT) TableResultT
func (a *WApp) NotesTableFilter(key TableKeyT, where string, seed *TableSeedT) (TableResultT, error)
func (a *WApp) NotesTableReconcile(surface, document string, keep []int)
func (a *WApp) NotesTableDisposeAll()
```

On `Missing`, the frontend re-seeds from the DOM (which is the source of truth
anyway) and retries once. This is the keystone: because eviction is always
recoverable, every other policy below can be aggressive without risking a
broken table.

**(c) Reconciliation, not disposal.** After rendering a surface the frontend
calls `NotesTableReconcile(surface, document, keep)` declaring which table
indices now exist. Go drops everything else for that surface. A missed cleanup
therefore self-heals on the next render, instead of accumulating — and re-render
is exactly the path most likely to leak, since it fires on every cell edit and
formula recompute. Explicit disposal still happens on file close, workspace
switch and `beforeunload` (alongside the existing `NotesLspStopAll()`), but
correctness does not depend on it.

**(d) Small tables are never retained.** Below a threshold (~2,000 cells, which
covers the large majority of markdown tables) the frontend sends the seed with
the query, and Go builds the SQLite table, answers, and drops it within the
call. The frontend knows the row count, so it decides up front whether to attach
the seed — one round trip either way. Retained memory therefore only ever comes
from genuinely large tables, where caching earns its keep. Both paths run
through `tablecore`, so semantics are identical; this is purely a retention
policy.

**(e) One shared database, with a budget.** Retained tables live as separate SQL
tables inside a *single* `:memory:` database rather than one database each.
Today `element_table` opens a `sql.DB` per element and pins it to one connection
(`db.go`, `SetMaxOpenConns(1)`), so N tables means N pools and N connections.
Sharing one database removes that overhead and makes memory measurable in one
place via `PRAGMA page_count * page_size`. Beyond a byte budget, least-recently
used tables are `DROP`ped; if the database still exceeds a hard ceiling it is
closed and reopened wholesale. Both are safe because of (b).

**(f) Idle unload after 10 minutes.** A table untouched for 10 minutes is
dropped, with the sweep running roughly once a minute. This is safe only
because of (b) — without recreate-on-miss, an idle timeout would leave a broken
table on screen.

Two caveats that shape the design:

- *It is a backstop, not a primary mechanism.* Reconcile (c) already reclaims on
  every re-render, and small tables (d) are never retained. What idle unload
  actually catches is the narrow case of a large table that was seeded, then
  neither re-rendered nor closed — "opened a big CSV, sorted it, left the app
  running".

- *The cost profile is inverted, so do not be aggressive.* Expiry costs a full
  re-seed over IPC, and the only tables ever retained are the large ones, so a
  short timeout penalises exactly the tables it targets. Ten minutes is
  comfortably past the point where a user is still interacting with a table,
  while still releasing memory long before the process would notice.

Critically, `DROP TABLE` on a `:memory:` database returns pages to SQLite's
freelist for reuse — it does **not** return memory to the OS. Idle unload is
therefore cosmetic unless the shared database is *closed* once the registry
empties, which is what actually releases the heap. `VACUUM` is the alternative
when other tables are still live and a large one has just been dropped.

The sweep goroutine starts when the first table is retained and stops when the
registry empties, so an idle session with no tables carries no background
goroutine.

Note that `runtime.AddCleanup`, used for the terminal element in
`element_table.go`, is no help here: the registry holds a strong reference, so
the finaliser would never run.

### 5. Mouse interactions

Terminal behaviour today (`mouse.go`):

| Target | Left | Right | Double-click |
|---|---|---|---|
| Heading | Toggle sort ASC/DESC, or switch column | Clear sort | — |
| Data cell | Copy cell to clipboard | Context menu (export CSV / Markdown) | SQL filter prompt |

Middle-click on a heading also clears the sort, on both surfaces.

Agreed Notes behaviour:

| Target | Left | Middle | Right | Double-click |
|---|---|---|---|---|
| Heading | **Toggle sort ASC/DESC** (terminal parity) | **Clear sort** | Context menu incl. "Clear sorting" and "SQL filter…" | — |
| Data cell | (unchanged — text selection) | — | Context menu incl. "SQL filter…" | **Edit cell (unchanged)** |

Deliberate divergences from the terminal, each with a reason:

- **Double-click keeps editing the cell.** Explicitly requested; the SQL filter
  moves to the right-click menu instead.
- **Left-click on a data cell does not copy to clipboard.** It would fight text
  selection and cell editing in a webview.
- **Ctrl+scroll is not adopted.** Notes tables scroll natively.

Sort direction is indicated in the heading with the same ↑ / ↓ glyphs the
terminal uses (`arrowGlyph`, `element_table.go` line 51), replacing the current
four-way FontAwesome icon set.

### 6. Deleted vs preserved

**Deleted** (`frontend/src/notes.js`):

- `setupTableSorting()` in full, lines 3697–3797 — including `applySort()`,
  `clearSort()`, `clearSortIcons()` and the six-item sort menu.
- `th.dataset.sortType` stamping.
- The four sort glyph code points (`0xf162`, `0xf886`, `0xf15d`, `0xf881`).
- `highlightEntireTable()` if it has no remaining caller.

**Preserved unchanged**:

- `setupTableColumnResizing()`, `ensureTableColGroup()`,
  `collectTableColumnWidths()`, `applyTableColumnWidths()` and the
  `GetNotesColumnWidths` / `SetNotesColumnWidths` bindings.
- `frontend/src/table-expressions.js` in full, plus `evaluateTableFormulasInPlace()`,
  `setupInteractiveTableCells()`, `runMarkdownTableFunction()` and the
  `data-formula` attribute.
- `applyNotesTableWordWrapMode()`, `state.markdownTableWordWrapMode` and all
  `.notes-table-wordwrap-on` CSS.
- `wrapTablesForHorizontalScroll()` and `.notes-table-scroll-wrap`.
- `row.dataset.originalSortOrder`, repurposed as the stable source-row index
  used to apply the permutation.

## Implementation status

Implemented in:

- `window/elements/element_table/tablecore/` — shared SQLite engine and registry
- `window/elements/element_table/` — terminal widget delegation
- `frontend_tables.go` — Notes Wails bindings and retention policy
- `frontend/src/notes.js` — four-surface integration, row permutation and menus
- `frontend/src/notes.css` — filtered rows and terminal-style sort indicators

The implementation includes per-table reconciliation, recreate-on-miss, small
table ephemeral builds, shared in-memory database retention, least-recently-used
eviction and the ten-minute idle sweep described above.

## Consequences

**Positive**

- One implementation of sort/filter semantics instead of two. Behaviour cannot
  drift because there is nothing to keep in step.
- Notes gains SQL filtering, which it has never had.
- Sorting becomes type-correct. The `parseFloat(x) || 0` bug — every
  non-numeric cell collapsing to zero — disappears.
- `element_table` currently has **zero tests**. Extracting a headless core makes
  the sort/filter rules unit-testable for the first time, for both consumers.

**Negative / risks**

- **Memory lifetime.** Retained tables are the main new failure mode. Mitigated
  by the layered design in "Wails bindings and lifetime": reproducible keys,
  recreate-on-miss, reconcile-on-render, no retention for small tables, a
  shared database under a byte budget, and a 10 minute idle unload. Residual
  risk is a single large table held for up to 10 minutes after last use, which
  is bounded and measurable rather than unbounded.
- **Snapshot staleness.** Editing a cell invalidates the Go-side table; the
  rebuild path must be reliable or sorting will silently act on stale values.
- **Sorting an editable table.** Hiding rows via a filter while the table is
  editable is a new interaction. Committing an edit to a filtered/sorted table
  must map back through the permutation to the correct source row.
- **SQL injection is the feature.** The filter is a raw `WHERE` fragment. The
  blast radius is a throwaway in-memory DB, but `go-sqlite3` supports `ATTACH`
  and `PRAGMA`, which can reach the filesystem. The filter must only ever come
  from explicit user input, and rejecting `;`, `ATTACH` and `PRAGMA` is cheap
  insurance.
- **Latency.** Sorting becomes an async IPC round-trip rather than a synchronous
  array sort. Imperceptible for typical tables. At scale the bottleneck is DOM
  reordering, not the bridge — see "Rejected: batching the permutation over
  IPC" for the measurements and the `DocumentFragment` mitigation.
- **Test churn.** Frontend tests asserting the sort menu must be rewritten
  against the binding plus DOM reordering.

## Resolved decisions

1. **Heading right-click opens a context menu** containing "Clear sorting" and
   "SQL filter…", rather than clearing the sort immediately. This is the one
   deliberate departure from terminal mouse parity, accepted because Notes needs
   a right-click affordance to reach the SQL filter and its existing table
   actions.
2. **Scope is all four surfaces**: CSV view, markdown preview, Jupyter view and
   the AI output panel.
3. **No filter persistence.** A filter resets on re-render and reload, matching
   the terminal.
4. **Package location** is `window/elements/element_table/tablecore`. A
   subpackage rather than `element_table` itself, so the core carries no
   dependency on `types.Renderer`, `cursor` or `golang.design/x/clipboard` —
   `clipboard` in particular requires `clipboard.Init()`, which would make
   headless unit tests awkward.

## Rejected: batching the permutation over IPC

Considered returning the row order in fixed-size blocks (e.g. 200 rows per call)
to avoid churning IPC. Rejected — it would make things slower, and the premise
does not apply to this design.

There is no row-by-row IPC to begin with. A sort is **one** call returning
**one** array of integers. Row *content* never crosses the bridge; only source
row indices do (see "Return a row permutation, not cell content" above).

Payload sizes for the whole permutation, as JSON:

| Rows | Approx. payload |
|---|---|
| 1,000 | ~4 KB |
| 10,000 | ~60 KB |
| 100,000 | ~700 KB |

Batching at 200 rows would turn a single 60 KB call into 50 round trips for a
10,000-row table, each with its own promise, JSON envelope and webview dispatch.
That is strictly worse. Chunking only pays when the payload is large *and* most
of it can be skipped — i.e. virtualised rendering, where only the visible window
is ever fetched. Notes renders every `<tr>` into the DOM, and a partial
permutation would mean a partially sorted table, which is simply incorrect.

The real cost at scale is **DOM reordering, not IPC**. Reordering N rows with
`tbody.appendChild(row)` in a loop risks a layout pass per row. The mitigation
is to batch on the DOM side instead: build a `DocumentFragment`, append the rows
in their new order, and attach it in a single operation. Filtering likewise
applies visibility changes within that one swap rather than row by row.

`limit` / `offset` remain in the `tablecore` API regardless, because the
terminal needs them for viewport pagination. So if a table ever appears that is
large enough to justify virtualised rendering in Notes, the engine already
supports it and this decision can be revisited without an API change. A
row-count threshold around 50,000 is the sensible point to reconsider.

