package tablecore

import (
	"sync"
	"testing"
	"time"
)

func newTestRegistry(t *testing.T) *Registry {
	t.Helper()

	r := NewRegistry()
	r.sweepInterval = 0 // no background goroutine; tests call Sweep directly
	t.Cleanup(r.Clear)

	return r
}

func key(index int) Key {
	return Key{Surface: "csv", Document: "/tmp/doc.csv", Index: index}
}

func putTestTable(t *testing.T, r *Registry, k Key) *Table {
	t.Helper()

	table, err := r.Put(k, []string{"n"}, [][]string{{"2"}, {"1"}})
	if err != nil {
		t.Fatalf("Put: %v", err)
	}
	return table
}

func TestRegistry_GetReturnsFalseForUnknownKey(t *testing.T) {
	r := newTestRegistry(t)

	if _, ok := r.Get(key(0)); ok {
		t.Fatal("Get on an empty registry reported a hit")
	}
}

func TestRegistry_PutThenGetRetainsSortState(t *testing.T) {
	r := newTestRegistry(t)

	table := putTestTable(t, r, key(0))
	table.ToggleSort(1)

	got, ok := r.Get(key(0))
	if !ok {
		t.Fatal("Get missed a table that was just stored")
	}
	if column, desc := got.SortState(); column != 1 || desc {
		t.Fatalf("sort state = (%d, %v), want (1, false) to survive retrieval", column, desc)
	}
}

func TestRegistry_PutReplacesExistingTable(t *testing.T) {
	r := newTestRegistry(t)

	first := putTestTable(t, r, key(0))
	first.ToggleSort(1)

	putTestTable(t, r, key(0))

	got, _ := r.Get(key(0))
	if column, _ := got.SortState(); column != Unsorted {
		t.Fatalf("sort column = %d, want a replaced table to start unsorted", column)
	}
	if r.Len() != 1 {
		t.Fatalf("Len = %d, want 1 after replacing the same key", r.Len())
	}
}

// Reconcile is the primary reclaim path: the frontend declares what still
// exists after a render and everything else for that document is dropped.
func TestRegistry_ReconcileDropsTablesNoLongerPresent(t *testing.T) {
	r := newTestRegistry(t)

	putTestTable(t, r, key(0))
	putTestTable(t, r, key(1))
	putTestTable(t, r, key(2))

	r.Reconcile("csv", "/tmp/doc.csv", []int{0, 2})

	if _, ok := r.Get(key(1)); ok {
		t.Fatal("Reconcile kept a table that was no longer declared")
	}
	if _, ok := r.Get(key(0)); !ok {
		t.Fatal("Reconcile dropped a table that was still declared")
	}
	if _, ok := r.Get(key(2)); !ok {
		t.Fatal("Reconcile dropped a table that was still declared")
	}
}

func TestRegistry_ReconcileLeavesOtherDocumentsAlone(t *testing.T) {
	r := newTestRegistry(t)

	putTestTable(t, r, key(0))
	other := Key{Surface: "preview", Document: "/tmp/notes.md", Index: 0}
	putTestTable(t, r, other)

	r.Reconcile("csv", "/tmp/doc.csv", nil)

	if _, ok := r.Get(other); !ok {
		t.Fatal("Reconcile dropped a table belonging to a different surface/document")
	}
}

func TestRegistry_SweepDropsIdleTables(t *testing.T) {
	r := newTestRegistry(t)

	now := time.Now()
	r.now = func() time.Time { return now }

	putTestTable(t, r, key(0))

	now = now.Add(DefaultIdleTimeout / 2)
	r.Sweep()
	if _, ok := r.Get(key(0)); !ok {
		t.Fatal("Sweep dropped a table that was still within the idle timeout")
	}

	now = now.Add(DefaultIdleTimeout * 2)
	r.Sweep()
	if _, ok := r.Get(key(0)); ok {
		t.Fatal("Sweep kept a table well past the idle timeout")
	}
}

// Reading a table must count as activity, otherwise a table in continuous use
// would be reaped mid-session.
func TestRegistry_GetRefreshesIdleTimer(t *testing.T) {
	r := newTestRegistry(t)

	now := time.Now()
	r.now = func() time.Time { return now }

	putTestTable(t, r, key(0))

	now = now.Add(DefaultIdleTimeout - time.Minute)
	if _, ok := r.Get(key(0)); !ok {
		t.Fatal("table expired earlier than expected")
	}

	now = now.Add(time.Minute * 2)
	r.Sweep()

	if _, ok := r.Get(key(0)); !ok {
		t.Fatal("Sweep ignored the refreshed idle timer")
	}
}

func TestRegistry_EvictsLeastRecentlyUsedOverCapacity(t *testing.T) {
	r := newTestRegistry(t)
	r.maxTables = 2

	now := time.Now()
	r.now = func() time.Time { return now }

	putTestTable(t, r, key(0))
	now = now.Add(time.Second)
	putTestTable(t, r, key(1))
	now = now.Add(time.Second)
	putTestTable(t, r, key(2))

	if r.Len() != 2 {
		t.Fatalf("Len = %d, want the registry capped at 2", r.Len())
	}
	if _, ok := r.Get(key(0)); ok {
		t.Fatal("expected the least recently used table to be evicted")
	}
}

// Closing the database when empty is what actually returns memory; dropping
// tables alone only recycles pages within SQLite.
func TestRegistry_ReleasesDatabaseWhenEmptied(t *testing.T) {
	r := newTestRegistry(t)

	putTestTable(t, r, key(0))
	if r.db == nil {
		t.Fatal("expected a database to be open while a table is retained")
	}

	r.Forget(key(0))

	if r.db != nil {
		t.Fatal("expected the database to be closed once the registry emptied")
	}
}

func TestRegistry_ReopensAfterBeingEmptied(t *testing.T) {
	r := newTestRegistry(t)

	putTestTable(t, r, key(0))
	r.Clear()
	putTestTable(t, r, key(0))

	if _, ok := r.Get(key(0)); !ok {
		t.Fatal("registry could not store a table after being emptied")
	}
}

// Small tables are answered without retention, so nothing accumulates.
func TestRegistry_BuildDoesNotRetain(t *testing.T) {
	r := newTestRegistry(t)

	table, release, err := r.Build(key(0), []string{"n"}, [][]string{{"2"}, {"1"}})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	defer release()

	table.ToggleSort(1)
	order, err := table.Order(0, 0)
	if err != nil {
		t.Fatalf("Order: %v", err)
	}
	assertOrder(t, order, 1, 0)

	if r.Len() != 0 {
		t.Fatalf("Len = %d, want Build to retain nothing", r.Len())
	}
}

func TestRegistry_ConcurrentAccessIsSafe(t *testing.T) {
	r := newTestRegistry(t)

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()

			k := key(i % 3)
			if _, err := r.Put(k, []string{"n"}, [][]string{{"1"}}); err != nil {
				t.Errorf("Put: %v", err)
				return
			}
			if table, ok := r.Get(k); ok {
				table.ToggleSort(1)
				_, _ = table.Order(0, 0)
			}
			r.Reconcile("csv", "/tmp/doc.csv", []int{0, 1, 2})
		}(i)
	}
	wg.Wait()
}
