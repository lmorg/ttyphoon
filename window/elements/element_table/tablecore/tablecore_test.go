package tablecore

import (
	"testing"
)

func newTestTable(t *testing.T, headings []string, rows [][]string) *Table {
	t.Helper()

	db, err := NewMemoryDB()
	if err != nil {
		t.Fatalf("NewMemoryDB: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	table, err := New(db, "test_table", headings, rows)
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	return table
}

func mustOrder(t *testing.T, table *Table) []int {
	t.Helper()

	order, err := table.Order(0, 0)
	if err != nil {
		t.Fatalf("Order: %v", err)
	}
	return order
}

func assertOrder(t *testing.T, got []int, want ...int) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("order = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order = %v, want %v", got, want)
		}
	}
}

// The terminal's left-click rule: the same column flips direction, a different
// column starts ascending.
func TestToggleSort_MatchesTerminalClickRule(t *testing.T) {
	table := newTestTable(t, []string{"a", "b"}, [][]string{{"1", "x"}, {"2", "y"}})

	table.ToggleSort(1)
	if col, desc := table.SortState(); col != 1 || desc {
		t.Fatalf("first click = (%d, %v), want (1, false)", col, desc)
	}

	table.ToggleSort(1)
	if col, desc := table.SortState(); col != 1 || !desc {
		t.Fatalf("second click = (%d, %v), want (1, true)", col, desc)
	}

	table.ToggleSort(2)
	if col, desc := table.SortState(); col != 2 || desc {
		t.Fatalf("new column = (%d, %v), want (2, false)", col, desc)
	}
}

func TestToggleSort_IgnoresOutOfRangeColumns(t *testing.T) {
	table := newTestTable(t, []string{"a"}, [][]string{{"1"}})

	table.ToggleSort(1)
	table.ToggleSort(99)

	if col, desc := table.SortState(); col != 1 || desc {
		t.Fatalf("state = (%d, %v), want the in-range sort to survive", col, desc)
	}
}

func TestClearSort_RestoresSourceOrder(t *testing.T) {
	table := newTestTable(t, []string{"n"}, [][]string{{"30"}, {"10"}, {"20"}})

	table.ToggleSort(1)
	assertOrder(t, mustOrder(t, table), 1, 2, 0)

	table.ClearSort()
	assertOrder(t, mustOrder(t, table), 0, 1, 2)
}

// A numeric column must sort by magnitude, not lexically: the JS implementation
// this replaces compared "100" < "20" as strings.
func TestOrder_NumericColumnSortsNumerically(t *testing.T) {
	table := newTestTable(t, []string{"n"}, [][]string{{"100"}, {"20"}, {"3"}})

	table.ToggleSort(1)
	assertOrder(t, mustOrder(t, table), 2, 1, 0)

	table.ToggleSort(1)
	assertOrder(t, mustOrder(t, table), 0, 1, 2)
}

func TestOrder_TextColumnSortsCaseInsensitively(t *testing.T) {
	table := newTestTable(t, []string{"s"}, [][]string{{"banana"}, {"Apple"}, {"cherry"}})

	table.ToggleSort(1)
	assertOrder(t, mustOrder(t, table), 1, 0, 2)
}

// Numeric detection samples the first data row only, matching the terminal. A
// non-numeric value in a numeric column sorts last, because SQLite orders text
// after numbers under NUMERIC affinity.
func TestOrder_ColumnTypeComesFromFirstRow(t *testing.T) {
	table := newTestTable(t, []string{"mixed"}, [][]string{{"10"}, {"9"}, {"abc"}})

	table.ToggleSort(1)

	assertOrder(t, mustOrder(t, table), 1, 0, 2)
}

func TestOrder_UnsortedFollowsSourceOrder(t *testing.T) {
	table := newTestTable(t, []string{"s"}, [][]string{{"c"}, {"a"}, {"b"}})

	assertOrder(t, mustOrder(t, table), 0, 1, 2)
}

func TestOrder_LimitAndOffsetPaginate(t *testing.T) {
	table := newTestTable(t, []string{"n"}, [][]string{{"1"}, {"2"}, {"3"}, {"4"}})

	order, err := table.Order(2, 1)
	if err != nil {
		t.Fatalf("Order: %v", err)
	}
	assertOrder(t, order, 1, 2)
}

func TestFilter_RestrictsAndComposesWithSort(t *testing.T) {
	table := newTestTable(t, []string{"n"}, [][]string{{"5"}, {"15"}, {"25"}})

	if err := table.SetFilter(`"n" > 10`); err != nil {
		t.Fatalf("SetFilter: %v", err)
	}
	assertOrder(t, mustOrder(t, table), 1, 2)

	table.ToggleSort(1)
	table.ToggleSort(1)
	assertOrder(t, mustOrder(t, table), 2, 1)
}

func TestFilter_EmptyStringClearsFilter(t *testing.T) {
	table := newTestTable(t, []string{"n"}, [][]string{{"5"}, {"15"}})

	if err := table.SetFilter(`"n" > 10`); err != nil {
		t.Fatalf("SetFilter: %v", err)
	}
	if err := table.SetFilter(""); err != nil {
		t.Fatalf("SetFilter clear: %v", err)
	}

	assertOrder(t, mustOrder(t, table), 0, 1)
}

// An invalid filter must leave the previous view usable rather than wedging the
// table in a state where every subsequent query fails.
func TestFilter_InvalidFilterIsRejectedAndPreviousRetained(t *testing.T) {
	table := newTestTable(t, []string{"n"}, [][]string{{"5"}, {"15"}})

	if err := table.SetFilter(`"n" > 10`); err != nil {
		t.Fatalf("SetFilter: %v", err)
	}

	if err := table.SetFilter(`this is not sql`); err == nil {
		t.Fatal("SetFilter accepted invalid SQL, want an error")
	}

	if got := table.Filter(); got != `"n" > 10` {
		t.Fatalf("Filter = %q, want the previous filter retained", got)
	}
	assertOrder(t, mustOrder(t, table), 1)
}

func TestFilter_RejectsStatementsThatCouldEscapeTheDatabase(t *testing.T) {
	table := newTestTable(t, []string{"n"}, [][]string{{"1"}})

	for _, filter := range []string{
		`1=1; DROP TABLE test_table`,
		`1=1 AND (SELECT 1 FROM pragma_database_list)`,
		"ATTACH DATABASE '/tmp/x.db' AS x",
		`1=1 AND load_extension('/tmp/evil.so') IS NULL`,
	} {
		if err := table.SetFilter(filter); err == nil {
			t.Fatalf("SetFilter(%q) was accepted, want rejection", filter)
		}
	}

	if got := table.Filter(); got != "" {
		t.Fatalf("Filter = %q, want empty after rejections", got)
	}
}

// The blocklist matches `pragma` as a prefix, so it must still be precise
// enough not to reject an ordinary column that merely starts with those letters.
func TestFilter_DoesNotRejectColumnsResemblingKeywords(t *testing.T) {
	table := newTestTable(t, []string{"pragmatic"}, [][]string{{"1"}, {"5"}})

	if err := table.SetFilter(`"pragmatic" > 2`); err != nil {
		t.Fatalf("SetFilter: %v", err)
	}

	assertOrder(t, mustOrder(t, table), 1)
}

func TestRows_ReturnsCellsInSortOrder(t *testing.T) {
	table := newTestTable(t, []string{"name", "qty"}, [][]string{
		{"pear", "2"},
		{"apple", "1"},
	})

	table.ToggleSort(1)

	rows, err := table.Rows(0, 0)
	if err != nil {
		t.Fatalf("Rows: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows = %v, want 2", rows)
	}
	if rows[0][0] != "apple" || rows[1][0] != "pear" {
		t.Fatalf("rows = %v, want apple before pear", rows)
	}
}

func TestNew_RaggedRowsArePaddedAndOverflowFolded(t *testing.T) {
	table := newTestTable(t, []string{"a", "b"}, [][]string{
		{"1"},
		{"2", "x", "y"},
	})

	rows, err := table.Rows(0, 0)
	if err != nil {
		t.Fatalf("Rows: %v", err)
	}

	if rows[0][1] != "" {
		t.Fatalf("short row = %v, want the missing cell padded", rows[0])
	}
	if rows[1][1] != "x y" {
		t.Fatalf("long row = %v, want overflow folded into the last column", rows[1])
	}
}

// SQLite rejects duplicate column names, which previously made such a CSV fail
// to load at all.
func TestNew_DuplicateHeadingsAreUsable(t *testing.T) {
	table := newTestTable(t, []string{"id", "id"}, [][]string{{"2", "b"}, {"1", "a"}})

	table.ToggleSort(2)
	assertOrder(t, mustOrder(t, table), 1, 0)

	if headings := table.Headings(); headings[1] != "id" {
		t.Fatalf("Headings = %v, want the original headings preserved", headings)
	}
}

func TestNew_RejectsEmptyHeadings(t *testing.T) {
	db, err := NewMemoryDB()
	if err != nil {
		t.Fatalf("NewMemoryDB: %v", err)
	}
	defer db.Close()

	if _, err := New(db, "t", nil, nil); err == nil {
		t.Fatal("New accepted empty headings, want an error")
	}
}

func TestCount_ReflectsFilter(t *testing.T) {
	table := newTestTable(t, []string{"n"}, [][]string{{"1"}, {"2"}, {"3"}})

	if count, err := table.Count(); err != nil || count != 3 {
		t.Fatalf("Count = %d, %v; want 3, nil", count, err)
	}

	if err := table.SetFilter(`"n" >= 2`); err != nil {
		t.Fatalf("SetFilter: %v", err)
	}

	if count, err := table.Count(); err != nil || count != 2 {
		t.Fatalf("filtered Count = %d, %v; want 2, nil", count, err)
	}
}

// Many tables share one database in the Notes registry, so they must not
// collide and dropping one must not disturb another.
func TestTablesShareADatabaseIndependently(t *testing.T) {
	db, err := NewMemoryDB()
	if err != nil {
		t.Fatalf("NewMemoryDB: %v", err)
	}
	defer db.Close()

	first, err := New(db, "first", []string{"n"}, [][]string{{"2"}, {"1"}})
	if err != nil {
		t.Fatalf("New first: %v", err)
	}
	second, err := New(db, "second", []string{"n"}, [][]string{{"9"}})
	if err != nil {
		t.Fatalf("New second: %v", err)
	}

	first.ToggleSort(1)
	assertOrder(t, mustOrder(t, first), 1, 0)

	if err := first.Drop(); err != nil {
		t.Fatalf("Drop: %v", err)
	}

	if count, err := second.Count(); err != nil || count != 1 {
		t.Fatalf("second Count = %d, %v; want 1, nil after dropping the other table", count, err)
	}
}

func TestDrop_IsSafeToRepeat(t *testing.T) {
	table := newTestTable(t, []string{"n"}, [][]string{{"1"}})

	if err := table.Drop(); err != nil {
		t.Fatalf("first Drop: %v", err)
	}
	if err := table.Drop(); err != nil {
		t.Fatalf("second Drop: %v", err)
	}
}
