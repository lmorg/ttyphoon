// Package tablecore is the headless table engine shared by the terminal table
// widget and the Notes frontend. It owns the SQLite-backed sort and filter
// semantics so that both surfaces behave identically by construction rather
// than by keeping two implementations in step.
//
// See adr/0028-go-backed-notes-table-sorting.md.
package tablecore

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

const driverName = "sqlite3"

// Unsorted is the sort column meaning "no explicit ordering": rows are returned
// in source order. Sort columns are otherwise 1-based.
const Unsorted = 0

const rowIDColumn = "rowid"

const (
	sqlCreateTable = `CREATE TABLE IF NOT EXISTS '%s' (%s);`
	sqlInsertRow   = `INSERT INTO '%s' VALUES (%s);`
	sqlDropTable   = `DROP TABLE IF EXISTS '%s';`
	sqlCount       = `SELECT count(*) FROM '%s' %s;`
)

// NewMemoryDB opens an in-memory database suitable for hosting tables.
//
// The pool is pinned to one connection because in-memory databases are
// per-connection: a second connection would see an empty database.
func NewMemoryDB() (*sql.DB, error) {
	db, err := sql.Open(driverName, ":memory:")
	if err != nil {
		return nil, fmt.Errorf("tablecore: cannot open in-memory database: %w", err)
	}

	db.SetMaxOpenConns(1)
	return db, nil
}

// Table is one tabular dataset together with its sort and filter state. It is
// safe for concurrent use.
type Table struct {
	mu       sync.Mutex
	db       *sql.DB
	name     string
	headings []string // as supplied by the caller
	columns  []string // SQL identifiers, de-duplicated
	isNumber []bool
	rowCount int

	filter   string
	sortCol  int // 1-based, or Unsorted
	sortDesc bool
}

// New creates a table inside db and populates it. Row order is preserved and
// becomes the source order that Order reports against.
func New(db *sql.DB, name string, headings []string, rows [][]string) (*Table, error) {
	if db == nil {
		return nil, fmt.Errorf("tablecore: nil database")
	}
	if len(headings) == 0 {
		return nil, fmt.Errorf("tablecore: cannot create table %q: no headings supplied", name)
	}

	t := &Table{
		db:       db,
		name:     sanitiseName(name),
		headings: append([]string(nil), headings...),
		columns:  uniqueColumns(headings),
		isNumber: make([]bool, len(headings)),
	}

	if err := t.create(); err != nil {
		return nil, err
	}

	if err := t.insertAll(rows); err != nil {
		_ = t.Drop()
		return nil, err
	}

	return t, nil
}

// Name returns the underlying SQL table name.
func (t *Table) Name() string { return t.name }

// Headings returns the column headings as supplied to New.
func (t *Table) Headings() []string {
	return append([]string(nil), t.headings...)
}

// Drop removes the table from its database. The database itself belongs to the
// caller and is not closed.
func (t *Table) Drop() error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if _, err := t.db.Exec(fmt.Sprintf(sqlDropTable, t.name)); err != nil {
		return fmt.Errorf("tablecore: cannot drop table %q: %w", t.name, err)
	}
	return nil
}

func (t *Table) create() error {
	var cols strings.Builder
	for i, column := range t.columns {
		if i > 0 {
			cols.WriteByte(',')
		}
		// NUMERIC affinity is what makes numeric columns sort numerically
		// despite every value being inserted as a string.
		fmt.Fprintf(&cols, `%s NUMERIC`, quoteIdent(column))
	}

	query := fmt.Sprintf(sqlCreateTable, t.name, cols.String())
	if _, err := t.db.Exec(query); err != nil {
		return fmt.Errorf("tablecore: cannot create table %q: %w\n%s", t.name, err, query)
	}

	return nil
}

func (t *Table) insertAll(rows [][]string) error {
	if len(rows) == 0 {
		return nil
	}

	tx, err := t.db.Begin()
	if err != nil {
		return fmt.Errorf("tablecore: cannot begin transaction: %w", err)
	}
	defer tx.Rollback()

	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(t.columns)), ",")
	stmt, err := tx.Prepare(fmt.Sprintf(sqlInsertRow, t.name, placeholders))
	if err != nil {
		return fmt.Errorf("tablecore: cannot prepare insert for %q: %w", t.name, err)
	}
	defer stmt.Close()

	for i, row := range rows {
		if _, err := stmt.Exec(toAny(t.normalise(row))...); err != nil {
			return fmt.Errorf("tablecore: cannot insert row %d into %q: %w", i, t.name, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("tablecore: cannot commit rows for %q: %w", t.name, err)
	}

	t.rowCount = len(rows)
	t.detectNumericColumns(rows[0])

	return nil
}

// normalise pads short rows and folds any overflow into the final column, which
// is what the terminal widget has always done with ragged CSV input.
func (t *Table) normalise(row []string) []string {
	n := len(t.columns)

	switch {
	case len(row) == n:
		return row

	case len(row) > n:
		out := append([]string(nil), row[:n]...)
		out[n-1] = strings.Join(row[n-1:], " ")
		return out

	default:
		out := make([]string, n)
		copy(out, row)
		return out
	}
}

// detectNumericColumns samples only the first data row, matching the terminal
// widget. A column that parses as a float sorts numerically; everything else
// sorts case-insensitively as text.
func (t *Table) detectNumericColumns(first []string) {
	row := t.normalise(first)
	for i := range t.columns {
		_, err := strconv.ParseFloat(strings.TrimSpace(row[i]), 64)
		t.isNumber[i] = err == nil
	}
}

// uniqueColumns de-duplicates headings. SQLite rejects duplicate column names,
// which would otherwise make a CSV with repeated headers fail to load at all.
func uniqueColumns(headings []string) []string {
	seen := make(map[string]int, len(headings))
	out := make([]string, len(headings))

	for i, heading := range headings {
		name := heading
		if strings.TrimSpace(name) == "" {
			name = fmt.Sprintf("column_%d", i+1)
		}

		if n, clash := seen[name]; clash {
			n++
			seen[name] = n
			name = fmt.Sprintf("%s_%d", name, n)
		} else {
			seen[name] = 0
		}

		out[i] = name
	}

	return out
}

func sanitiseName(name string) string {
	return strings.ReplaceAll(name, "'", "_")
}

func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

func toAny(s []string) []any {
	a := make([]any, len(s))
	for i := range s {
		a[i] = s[i]
	}
	return a
}
