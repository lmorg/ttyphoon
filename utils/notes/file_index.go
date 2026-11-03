package notes

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/lmorg/ttyphoon/utils/file/dirwatch"
)

type RootScanResult struct {
	Files     []string
	Truncated bool
	Err       error
	Entries   int
}

type FileIndexSnapshot struct {
	Root      string   `json:"root"`
	Files     []string `json:"files"`
	Version   uint64   `json:"version"`
	Revision  uint64   `json:"revision"`
	State     string   `json:"state"`
	Stale     bool     `json:"stale"`
	Error     string   `json:"error"`
	Truncated bool     `json:"truncated"`
}

type FileIndexEvent struct {
	Root        string   `json:"root"`
	Version     uint64   `json:"version"`
	Revision    uint64   `json:"revision"`
	BaseVersion uint64   `json:"baseVersion"`
	Added       []string `json:"added"`
	Removed     []string `json:"removed"`
	State       string   `json:"state"`
	Stale       bool     `json:"stale"`
	Truncated   bool     `json:"truncated"`
	Error       string   `json:"error"`
}

type rootIndexEntry struct {
	snapshot        FileIndexSnapshot
	cancel          context.CancelFunc
	generation      uint64
	startup         bool
	pending         bool
	used            uint64
	watchCancel     context.CancelFunc
	watchReady      chan struct{}
	watchGeneration uint64
	changes         map[string]bool
	changeSequence  uint64
	changeTimer     *time.Timer
	rescan          bool
	entries         int
}

type rootScanJob struct {
	ctx         context.Context
	root        string
	entry       *rootIndexEntry
	generation  uint64
	paths       []string
	base        []string
	baseEntries int
}

type FileIndex struct {
	mu       sync.Mutex
	ctx      context.Context
	cancel   context.CancelFunc
	entries  map[string]*rootIndexEntry
	active   map[string]bool
	clock    uint64
	revision uint64
	capacity int
	limit    int
	running  int
	queue    []rootScanJob
	scan     func(context.Context, string) RootScanResult
	onChange func(FileIndexEvent)
	workers  sync.WaitGroup
	closed   bool
	watch    func(context.Context, string, func(string) bool, func(dirwatch.Change), func(error))
	watchers sync.WaitGroup
}

func NewFileIndex(parent context.Context, capacity, concurrency int, scan func(context.Context, string) RootScanResult, onChange func(FileIndexEvent)) *FileIndex {
	if parent == nil {
		parent = context.Background()
	}
	if capacity < 1 {
		capacity = 16
	}
	if concurrency < 1 {
		concurrency = 2
	}
	ctx, cancel := context.WithCancel(parent)
	return &FileIndex{
		ctx: ctx, cancel: cancel, capacity: capacity, limit: concurrency,
		entries: make(map[string]*rootIndexEntry), active: make(map[string]bool),
		scan: scan, onChange: onChange,
	}
}

func indexRoot(path string) string {
	if path == "" {
		return ""
	}
	root, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}
	return filepath.Clean(root)
}

func (index *FileIndex) entryLocked(root string) *rootIndexEntry {
	entry := index.entries[root]
	if entry == nil {
		entry = &rootIndexEntry{snapshot: FileIndexSnapshot{Root: root, Files: []string{}, State: "scanning", Stale: true}}
		index.entries[root] = entry
	}
	index.clock++
	entry.used = index.clock
	return entry
}

func (index *FileIndex) Activate(roots []string) {
	index.mu.Lock()
	defer index.mu.Unlock()
	if index.closed {
		return
	}
	next := make(map[string]bool)
	for _, path := range roots {
		if root := indexRoot(path); root != "" {
			next[root] = true
		}
	}
	for root := range index.active {
		if next[root] {
			continue
		}
		entry := index.entries[root]
		index.stopWatchLocked(entry)
		entry.snapshot.Stale = true
		entry.pending = false
		if entry.cancel != nil && !entry.startup {
			entry.cancel()
			entry.generation++
		}
	}
	previous := index.active
	index.active = next
	for root := range next {
		entry := index.entryLocked(root)
		if previous[root] {
			continue
		}
		index.startWatchLocked(root, entry)
		if entry.cancel != nil {
			entry.pending = !entry.startup || index.watch != nil
		} else {
			index.startLocked(root, entry, false)
		}
	}
	index.pruneLocked()
}

func (index *FileIndex) Prewarm(roots []string) {
	index.mu.Lock()
	defer index.mu.Unlock()
	if index.closed {
		return
	}
	for _, path := range roots {
		root := indexRoot(path)
		if root == "" || index.active[root] {
			continue
		}
		entry := index.entryLocked(root)
		if entry.cancel == nil && entry.snapshot.Version == 0 {
			index.startLocked(root, entry, true)
		}
	}
}

func (index *FileIndex) Refresh(roots []string) {
	index.mu.Lock()
	defer index.mu.Unlock()
	if index.closed {
		return
	}
	for _, path := range roots {
		root := indexRoot(path)
		if !index.active[root] {
			continue
		}
		entry := index.entryLocked(root)
		if entry.cancel != nil {
			entry.pending = true
		} else {
			index.startLocked(root, entry, false)
		}
	}
}

func (index *FileIndex) Snapshots(roots []string) []FileIndexSnapshot {
	index.mu.Lock()
	defer index.mu.Unlock()
	snapshots := make([]FileIndexSnapshot, 0, len(roots))
	for _, path := range roots {
		root := indexRoot(path)
		if root == "" {
			continue
		}
		entry := index.entries[root]
		if entry == nil {
			continue
		}
		index.clock++
		entry.used = index.clock
		snapshot := entry.snapshot
		snapshot.Files = append([]string{}, snapshot.Files...)
		snapshots = append(snapshots, snapshot)
	}
	return snapshots
}

func (index *FileIndex) startLocked(root string, entry *rootIndexEntry, startup bool) {
	ctx, cancel := context.WithCancel(index.ctx)
	entry.cancel = cancel
	entry.startup = startup
	entry.pending = false
	entry.generation++
	generation := entry.generation
	entry.snapshot.State = "scanning"
	if entry.snapshot.Version > 0 {
		entry.snapshot.State = "refreshing"
	}
	entry.snapshot.Stale = true
	entry.snapshot.Error = ""
	index.revision++
	entry.snapshot.Revision = index.revision
	index.workers.Add(1)
	index.queue = append(index.queue, rootScanJob{ctx: ctx, root: root, entry: entry, generation: generation})
	index.dispatchLocked()
}

func (index *FileIndex) dispatchLocked() {
	for index.running < index.limit && len(index.queue) > 0 {
		position := 0
		for queued, job := range index.queue {
			if index.active[job.root] {
				position = queued
				break
			}
		}
		job := index.queue[position]
		index.queue = append(index.queue[:position], index.queue[position+1:]...)
		index.running++
		go index.run(job)
	}
}

func (index *FileIndex) run(job rootScanJob) {
	defer index.workers.Done()
	ctx, root, entry, generation := job.ctx, job.root, job.entry, job.generation
	index.mu.Lock()
	ready := entry.watchReady
	changeSequence := entry.changeSequence
	index.mu.Unlock()
	if ready != nil {
		select {
		case <-ready:
		case <-ctx.Done():
		}
	}
	var result RootScanResult
	if ctx.Err() == nil {
		if job.paths != nil {
			result = reconcileIndexPaths(ctx, root, job.base, job.paths, job.baseEntries)
		} else {
			result = index.scan(ctx, root)
		}
	}
	index.mu.Lock()
	index.running--
	cancel := entry.cancel
	entry.cancel = nil
	previousFiles := entry.snapshot.Files
	previousVersion := entry.snapshot.Version
	dirty := index.active[root] && changeSequence != entry.changeSequence
	if !index.closed && ctx.Err() == nil && generation == entry.generation {
		if dirty {
			entry.pending = index.active[root]
		} else if result.Err != nil {
			entry.snapshot.State = "error"
			entry.snapshot.Error = result.Err.Error()
		} else {
			entry.snapshot.Files = append([]string{}, result.Files...)
			sort.Strings(entry.snapshot.Files)
			if entry.snapshot.Version == 0 || !equalIndexFiles(entry.snapshot.Files, previousFiles) {
				entry.snapshot.Version++
			}
			entry.snapshot.Truncated = result.Truncated
			entry.entries = result.Entries
			entry.snapshot.State = "ready"
			if result.Truncated {
				entry.snapshot.State = "truncated"
			}
			entry.snapshot.Stale = !index.active[root]
		}
	} else {
		entry.snapshot.Stale = true
		entry.snapshot.State = "ready"
		if entry.snapshot.Truncated {
			entry.snapshot.State = "truncated"
		}
	}
	cancel()
	index.revision++
	entry.snapshot.Revision = index.revision
	if !index.closed && entry.pending && index.active[root] {
		entry.changes = nil
		entry.rescan = false
		if entry.changeTimer != nil {
			entry.changeTimer.Stop()
			entry.changeTimer = nil
		}
		index.startLocked(root, entry, false)
	}
	event := indexEvent(entry.snapshot, previousVersion, previousFiles)
	index.pruneLocked()
	index.dispatchLocked()
	closed := index.closed
	index.mu.Unlock()
	if !closed && index.onChange != nil {
		index.onChange(event)
	}
}

func (index *FileIndex) pruneLocked() {
	for len(index.entries) > index.capacity {
		oldest := ""
		var used uint64
		for root, entry := range index.entries {
			if index.active[root] || entry.cancel != nil {
				continue
			}
			if oldest == "" || entry.used < used {
				oldest, used = root, entry.used
			}
		}
		if oldest == "" {
			return
		}
		delete(index.entries, oldest)
	}
}

func (index *FileIndex) Close() {
	index.mu.Lock()
	index.closed = true
	for _, entry := range index.entries {
		index.stopWatchLocked(entry)
	}
	index.cancel()
	index.mu.Unlock()
	index.workers.Wait()
	index.watchers.Wait()
}

func ScanIndexRoot(ctx context.Context, root string) RootScanResult {
	if err := ctx.Err(); err != nil {
		return RootScanResult{Err: err}
	}
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return RootScanResult{Files: []string{}}
	}
	result := ScanFiles(ctx, FileContext{PathProjectRoot: root})
	files := make([]string, 0, len(result.Files))
	for _, file := range result.Files {
		files = append(files, strings.TrimPrefix(file, "$PROJECT"+string(filepath.Separator)))
	}
	return RootScanResult{Files: files, Truncated: result.Truncated, Err: result.Err, Entries: result.Entries}
}
