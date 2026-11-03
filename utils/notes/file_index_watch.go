package notes

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/utils/file/dirwatch"
)

func (index *FileIndex) EnableWatching() {
	index.mu.Lock()
	defer index.mu.Unlock()
	if index.closed || index.watch != nil {
		return
	}
	index.watch = dirwatch.Run
	for root := range index.active {
		entry := index.entries[root]
		index.startWatchLocked(root, entry)
		entry.pending = entry.cancel != nil
		if entry.cancel == nil {
			index.startLocked(root, entry, false)
		}
	}
}

func ignoredIndexPath(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return true
	}
	if relative == "." {
		return false
	}
	for _, part := range strings.Split(relative, string(filepath.Separator)) {
		if config.Config.Notes.ExcludeDirectories[part] || isSystemFileName(part) {
			return true
		}
	}
	return false
}

func (index *FileIndex) startWatchLocked(root string, entry *rootIndexEntry) {
	if index.watch == nil || entry.watchCancel != nil || index.closed {
		return
	}
	ctx, cancel := context.WithCancel(index.ctx)
	entry.watchCancel = cancel
	entry.watchReady = make(chan struct{})
	entry.watchGeneration++
	generation := entry.watchGeneration
	ready := entry.watchReady
	index.watchers.Add(1)
	go func() {
		defer index.watchers.Done()
		index.watch(ctx, root, func(path string) bool { return ignoredIndexPath(root, path) },
			func(change dirwatch.Change) { index.recordWatchChange(root, generation, change) },
			func(err error) { close(ready) })
	}()
}

func (index *FileIndex) stopWatchLocked(entry *rootIndexEntry) {
	entry.watchGeneration++
	if entry.watchCancel != nil {
		entry.watchCancel()
		entry.watchCancel = nil
	}
	entry.watchReady = nil
	if entry.changeTimer != nil {
		entry.changeTimer.Stop()
		entry.changeTimer = nil
	}
	entry.changes = nil
	entry.rescan = false
}

func (index *FileIndex) recordWatchChange(root string, generation uint64, change dirwatch.Change) {
	index.mu.Lock()
	defer index.mu.Unlock()
	entry := index.entries[root]
	if index.closed || !index.active[root] || entry == nil || entry.watchGeneration != generation {
		return
	}
	index.recordChangeLocked(root, entry, change)
}

func (index *FileIndex) recordChangeLocked(root string, entry *rootIndexEntry, change dirwatch.Change) {
	if change.Poll && entry.cancel != nil {
		return
	}
	if !change.Rescan && ignoredIndexPath(root, change.Path) {
		return
	}
	entry.changeSequence++
	if entry.rescan || change.Rescan || len(entry.changes) >= 256 {
		entry.rescan = true
		entry.changes = nil
	} else {
		if entry.changes == nil {
			entry.changes = make(map[string]bool)
		}
		entry.changes[change.Path] = true
	}
	if entry.changeTimer == nil {
		generation := entry.watchGeneration
		entry.changeTimer = time.AfterFunc(200*time.Millisecond, func() { index.flushChanges(root, generation) })
	}
}

func (index *FileIndex) flushChanges(root string, generation uint64) {
	index.mu.Lock()
	var event *FileIndexEvent
	defer func() {
		index.mu.Unlock()
		if event != nil && index.onChange != nil {
			index.onChange(*event)
		}
	}()
	entry := index.entries[root]
	if index.closed || entry == nil || !index.active[root] || generation != entry.watchGeneration {
		return
	}
	entry.changeTimer = nil
	if len(entry.changes) == 0 && !entry.rescan {
		return
	}
	if entry.cancel != nil {
		entry.pending = true
		return
	}
	paths := make([]string, 0, len(entry.changes))
	for path := range entry.changes {
		paths = append(paths, path)
	}
	full := entry.rescan || entry.snapshot.Truncated || entry.snapshot.Version == 0 || entry.snapshot.State == "error"
	entry.changes = nil
	entry.rescan = false
	if full {
		index.startLocked(root, entry, false)
		started := indexEvent(entry.snapshot, entry.snapshot.Version, entry.snapshot.Files)
		event = &started
		return
	}
	ctx, cancel := context.WithCancel(index.ctx)
	entry.cancel = cancel
	entry.startup = false
	entry.pending = false
	entry.generation++
	index.workers.Add(1)
	index.queue = append(index.queue, rootScanJob{
		ctx: ctx, root: root, entry: entry, generation: entry.generation,
		paths: paths, base: append([]string{}, entry.snapshot.Files...), baseEntries: entry.entries,
	})
	index.dispatchLocked()
}

func (index *FileIndex) NotifyPaths(paths ...string) {
	index.mu.Lock()
	defer index.mu.Unlock()
	if index.closed {
		return
	}
	for root := range index.active {
		for _, path := range paths {
			path = indexRoot(path)
			if !ignoredIndexPath(root, path) {
				index.recordChangeLocked(root, index.entries[root], dirwatch.Change{Path: path})
			}
		}
	}
}

func reconcileIndexPaths(ctx context.Context, root string, base, paths []string, entries int) RootScanResult {
	files := make(map[string]bool, len(base))
	for _, file := range base {
		files[file] = true
	}
	sort.Strings(paths)
	for _, path := range paths {
		if ctx.Err() != nil {
			return RootScanResult{Err: ctx.Err()}
		}
		if ignoredIndexPath(root, path) {
			continue
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return RootScanResult{Err: err}
		}
		for file := range files {
			if relative == "." || file == relative || strings.HasPrefix(file, relative+string(filepath.Separator)) {
				delete(files, file)
				entries--
			}
		}
		info, err := os.Lstat(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return RootScanResult{Err: err}
		}
		if info.IsDir() {
			if relative == "." {
				return ScanIndexRoot(ctx, root)
			}
			result := ScanIndexRoot(ctx, path)
			if result.Err != nil {
				return RootScanResult{Err: result.Err}
			}
			if result.Truncated {
				return ScanIndexRoot(ctx, root)
			}
			entries += result.Entries
			for _, file := range result.Files {
				joined := filepath.Join(relative, file)
				if !ignoredIndexPath(root, filepath.Join(root, joined)) {
					files[joined] = true
				}
			}
		} else {
			files[relative] = true
			entries++
		}
		if entries >= MAX_FILES || len(files) >= MAX_FILES {
			return ScanIndexRoot(ctx, root)
		}
	}
	result := RootScanResult{Files: make([]string, 0, len(files)), Entries: max(entries, len(files))}
	for file := range files {
		result.Files = append(result.Files, file)
	}
	sort.Strings(result.Files)
	return result
}

func equalIndexFiles(first, second []string) bool {
	if len(first) != len(second) {
		return false
	}
	for position, file := range first {
		if second[position] != file {
			return false
		}
	}
	return true
}

func indexEvent(snapshot FileIndexSnapshot, previousVersion uint64, previousFiles []string) FileIndexEvent {
	event := FileIndexEvent{Root: snapshot.Root, Version: snapshot.Version, Revision: snapshot.Revision,
		BaseVersion: previousVersion, State: snapshot.State, Stale: snapshot.Stale, Truncated: snapshot.Truncated, Error: snapshot.Error}
	previous := make(map[string]bool, len(previousFiles))
	for _, file := range previousFiles {
		previous[file] = true
	}
	for _, file := range snapshot.Files {
		if !previous[file] {
			event.Added = append(event.Added, file)
		}
		delete(previous, file)
	}
	for file := range previous {
		event.Removed = append(event.Removed, file)
	}
	sort.Strings(event.Removed)
	return event
}
