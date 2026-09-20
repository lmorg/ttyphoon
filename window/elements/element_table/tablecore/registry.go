package tablecore

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"
	"time"
)

// Defaults for the Notes registry. See adr/0028-go-backed-notes-table-sorting.md.
const (
	// DefaultIdleTimeout drops a table that has not been queried recently.
	DefaultIdleTimeout = 10 * time.Minute

	// DefaultSweepInterval is how often idle tables are reaped.
	DefaultSweepInterval = time.Minute

	// DefaultMaxTables caps how many tables are retained at once.
	DefaultMaxTables = 24
)

// Key addresses a table by something the caller can always reconstruct, so that
// losing server-side state is recoverable rather than fatal.
type Key struct {
	Surface  string `json:"surface"`
	Document string `json:"document"`
	Index    int    `json:"index"`
}

func (k Key) String() string {
	return fmt.Sprintf("%s\x00%s\x00%d", k.Surface, k.Document, k.Index)
}

func (k Key) tableName() string {
	name := fmt.Sprintf("notes_%s_%s_%d", k.Surface, k.Document, k.Index)
	name = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_':
			return r
		default:
			return '_'
		}
	}, name)

	return name
}

type entry struct {
	key      Key
	table    *Table
	lastUsed time.Time
}

// Registry holds the tables Notes is currently working with, inside a single
// shared in-memory database.
//
// Retention is deliberately best-effort: callers are expected to cope with a
// table having been evicted by rebuilding it, which is what lets the registry
// reclaim memory aggressively without breaking the UI.
type Registry struct {
	mu      sync.Mutex
	entries map[string]*entry

	// db is opened lazily and closed again whenever the registry empties;
	// dropping tables alone would return pages to SQLite's freelist rather
	// than to the OS.
	db *sql.DB

	idleTimeout   time.Duration
	sweepInterval time.Duration
	maxTables     int

	sweeping bool
	stop     chan struct{}

	now func() time.Time
}

// NewRegistry returns an empty registry. No database is opened until the first
// table is stored.
func NewRegistry() *Registry {
	return &Registry{
		entries:       make(map[string]*entry),
		idleTimeout:   DefaultIdleTimeout,
		sweepInterval: DefaultSweepInterval,
		maxTables:     DefaultMaxTables,
		now:           time.Now,
	}
}

// Get returns the table for key, or false when it is not held. A hit refreshes
// the idle timer.
func (r *Registry) Get(key Key) (*Table, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()

	e, ok := r.entries[key.String()]
	if !ok {
		return nil, false
	}

	e.lastUsed = r.now()
	return e.table, true
}

// Put builds a table from the supplied data and retains it under key, replacing
// any table already held there.
func (r *Registry) Put(key Key, headings []string, rows [][]string) (*Table, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if err := r.openLocked(); err != nil {
		return nil, err
	}

	if existing, ok := r.entries[key.String()]; ok {
		_ = existing.table.Drop()
		delete(r.entries, key.String())
	}

	table, err := New(r.db, key.tableName(), headings, rows)
	if err != nil {
		r.closeIfEmptyLocked()
		return nil, err
	}

	r.entries[key.String()] = &entry{key: key, table: table, lastUsed: r.now()}
	r.evictOverCapacityLocked()
	r.startSweepLocked()

	return table, nil
}

// Build returns a table for key without retaining it, for datasets small enough
// that rebuilding per query is cheaper than holding a database open.
func (r *Registry) Build(key Key, headings []string, rows [][]string) (*Table, func(), error) {
	db, err := NewMemoryDB()
	if err != nil {
		return nil, nil, err
	}

	table, err := New(db, key.tableName(), headings, rows)
	if err != nil {
		_ = db.Close()
		return nil, nil, err
	}

	return table, func() { _ = db.Close() }, nil
}

// Reconcile drops every table held for a surface and document whose index is not
// in keep. Callers declare what currently exists rather than tracking what to
// free, so a missed cleanup self-heals on the next render.
func (r *Registry) Reconcile(surface, document string, keep []int) {
	retain := make(map[int]struct{}, len(keep))
	for _, index := range keep {
		retain[index] = struct{}{}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for id, e := range r.entries {
		if e.key.Surface != surface || e.key.Document != document {
			continue
		}
		if _, ok := retain[e.key.Index]; ok {
			continue
		}

		_ = e.table.Drop()
		delete(r.entries, id)
	}

	r.closeIfEmptyLocked()
}

// Forget drops a single table if it is held.
func (r *Registry) Forget(key Key) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if e, ok := r.entries[key.String()]; ok {
		_ = e.table.Drop()
		delete(r.entries, key.String())
	}

	r.closeIfEmptyLocked()
}

// Clear drops every table and releases the database.
func (r *Registry) Clear() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for id, e := range r.entries {
		_ = e.table.Drop()
		delete(r.entries, id)
	}

	r.closeIfEmptyLocked()
}

// Len reports how many tables are currently retained.
func (r *Registry) Len() int {
	r.mu.Lock()
	defer r.mu.Unlock()

	return len(r.entries)
}

// Sweep drops tables idle for longer than the timeout. It is called on a timer
// but is exported so callers (and tests) can force a pass.
func (r *Registry) Sweep() {
	r.mu.Lock()
	defer r.mu.Unlock()

	cutoff := r.now().Add(-r.idleTimeout)
	for id, e := range r.entries {
		if e.lastUsed.After(cutoff) {
			continue
		}

		_ = e.table.Drop()
		delete(r.entries, id)
	}

	r.closeIfEmptyLocked()
}

func (r *Registry) evictOverCapacityLocked() {
	for len(r.entries) > r.maxTables {
		var (
			oldestID string
			oldest   time.Time
		)
		for id, e := range r.entries {
			if oldestID == "" || e.lastUsed.Before(oldest) {
				oldestID, oldest = id, e.lastUsed
			}
		}
		if oldestID == "" {
			return
		}

		_ = r.entries[oldestID].table.Drop()
		delete(r.entries, oldestID)
	}
}

func (r *Registry) openLocked() error {
	if r.db != nil {
		return nil
	}

	handle, err := NewMemoryDB()
	if err != nil {
		return err
	}

	r.db = handle
	return nil
}

// closeIfEmptyLocked releases the database once nothing is held. DROP TABLE only
// returns pages to SQLite's freelist, so closing is what actually hands memory
// back to the OS.
func (r *Registry) closeIfEmptyLocked() {
	if len(r.entries) > 0 || r.db == nil {
		return
	}

	_ = r.db.Close()
	r.db = nil
	r.stopSweepLocked()
}

func (r *Registry) startSweepLocked() {
	if r.sweeping || r.sweepInterval <= 0 {
		return
	}

	r.sweeping = true
	r.stop = make(chan struct{})

	go func(interval time.Duration, stop <-chan struct{}) {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				r.Sweep()
			}
		}
	}(r.sweepInterval, r.stop)
}

func (r *Registry) stopSweepLocked() {
	if !r.sweeping {
		return
	}

	close(r.stop)
	r.sweeping = false
	r.stop = nil
}
