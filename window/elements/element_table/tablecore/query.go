package tablecore

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	sqlSelectAlpha   = `SELECT %s FROM '%s' %s ORDER BY lower(%s) %s LIMIT %d OFFSET %d;`
	sqlSelectNumeric = `SELECT %s FROM '%s' %s ORDER BY %s %s LIMIT %d OFFSET %d;`
)

var orderDir = map[bool]string{
	false: "ASC",
	true:  "DESC",
}

// ATTACH and friends would let a filter reach the filesystem from what is
// otherwise a throwaway in-memory database. `pragma` is matched as a prefix
// because SQLite also exposes pragmas as `pragma_*` table-valued functions,
// which a word-boundary match would miss.
var unsafeFilterWords = regexp.MustCompile(`(?i)\battach\b|\bdetach\b|\bvacuum\b|\bpragma[\s_]|\bload_extension\b`)

// ToggleSort applies the terminal widget's left-click rule: clicking the column
// already being sorted flips the direction, any other column starts ascending.
// Columns are 1-based; out of range values are ignored.
func (t *Table) ToggleSort(column int) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if column < 1 || column > len(t.columns) {
		return
	}

	if t.sortCol == column {
		t.sortDesc = !t.sortDesc
		return
	}

	t.sortCol = column
	t.sortDesc = false
}

// SetSort selects a column and direction outright, without the toggle rule.
func (t *Table) SetSort(column int, desc bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if column < 1 || column > len(t.columns) {
		t.sortCol = Unsorted
		t.sortDesc = false
		return
	}

	t.sortCol = column
	t.sortDesc = desc
}

// ClearSort restores source order, matching a right-click on a terminal heading.
func (t *Table) ClearSort() {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.sortCol = Unsorted
	t.sortDesc = false
}

// SortState reports the current sort column (1-based, or Unsorted) and direction.
func (t *Table) SortState() (column int, desc bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.sortCol, t.sortDesc
}

// Filter returns the current WHERE fragment, without the WHERE keyword.
func (t *Table) Filter() string {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.filter
}

// SetFilter installs a raw SQL WHERE fragment; an empty string clears it.
//
// The fragment is validated before being kept, so a bad filter leaves the
// previous view intact rather than wedging the table in an unqueryable state.
func (t *Table) SetFilter(where string) error {
	where = strings.TrimSpace(where)

	if err := checkFilterSafe(where); err != nil {
		return err
	}

	t.mu.Lock()
	defer t.mu.Unlock()

	previous := t.filter
	t.filter = where

	if _, err := t.countLocked(); err != nil {
		t.filter = previous
		return fmt.Errorf("tablecore: invalid filter: %w", err)
	}

	return nil
}

// Order returns the source row indices matching the current filter, in the
// current sort order. A limit of zero or less returns every matching row.
//
// Returning indices rather than cell content is what lets a caller reorder rows
// it has already rendered without disturbing their contents.
func (t *Table) Order(limit, offset int) ([]int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	query := t.selectLocked(rowIDColumn, limit, offset)
	rows, err := t.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("tablecore: cannot query order: %w\nSQL: %s", err, query)
	}
	defer rows.Close()

	out := make([]int, 0, t.rowCount)
	for rows.Next() {
		var rowID int64
		if err := rows.Scan(&rowID); err != nil {
			return nil, fmt.Errorf("tablecore: cannot scan rowid: %w", err)
		}
		// Rows are inserted in source order and never deleted, so SQLite's
		// implicit rowid counts 1..N in that same order.
		out = append(out, int(rowID)-1)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("tablecore: cannot read order: %w", err)
	}

	return out, nil
}

// Rows returns cell values matching the current filter, in the current sort
// order. A limit of zero or less returns every matching row.
func (t *Table) Rows(limit, offset int) ([][]string, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	query := t.selectLocked("*", limit, offset)
	rows, err := t.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("tablecore: cannot query rows: %w\nSQL: %s", err, query)
	}
	defer rows.Close()

	var out [][]string
	for rows.Next() {
		row := make([]string, len(t.columns))
		if err := rows.Scan(scanTargets(row)...); err != nil {
			return nil, fmt.Errorf("tablecore: cannot scan row: %w", err)
		}
		out = append(out, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("tablecore: cannot read rows: %w", err)
	}

	return out, nil
}

// Count returns the number of rows matching the current filter.
func (t *Table) Count() (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	return t.countLocked()
}

func (t *Table) countLocked() (int, error) {
	var count int
	query := fmt.Sprintf(sqlCount, t.name, t.whereLocked())
	if err := t.db.QueryRow(query).Scan(&count); err != nil {
		return 0, fmt.Errorf("cannot count rows: %w", err)
	}

	return count, nil
}

func (t *Table) whereLocked() string {
	if t.filter == "" {
		return ""
	}

	return "WHERE " + t.filter
}

func (t *Table) selectLocked(projection string, limit, offset int) string {
	template := sqlSelectNumeric
	orderBy := quoteIdent(rowIDColumn)

	if t.sortCol != Unsorted {
		orderBy = quoteIdent(t.columns[t.sortCol-1])
		if !t.isNumber[t.sortCol-1] {
			template = sqlSelectAlpha
		}
	}

	// SQLite treats a negative LIMIT as unbounded.
	if limit <= 0 {
		limit = -1
		offset = 0
	}
	if offset < 0 {
		offset = 0
	}

	return fmt.Sprintf(template, projection, t.name, t.whereLocked(), orderBy, orderDir[t.sortDesc], limit, offset)
}

func checkFilterSafe(where string) error {
	if where == "" {
		return nil
	}

	if strings.Contains(where, ";") {
		return fmt.Errorf("tablecore: filter may not contain ';'")
	}

	if word := unsafeFilterWords.FindString(where); word != "" {
		return fmt.Errorf("tablecore: filter may not contain %q", word)
	}

	return nil
}

func scanTargets(row []string) []any {
	targets := make([]any, len(row))
	for i := range row {
		targets[i] = &row[i]
	}
	return targets
}
