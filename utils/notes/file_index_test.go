package notes

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/utils/file/dirwatch"
)

func TestFileIndexReturnsSnapshotWhileScanIsBlocked(t *testing.T) {
	started := make(chan struct{})
	finish := make(chan struct{})
	events := make(chan FileIndexEvent, 4)
	index := NewFileIndex(context.Background(), 4, 1, func(ctx context.Context, root string) RootScanResult {
		close(started)
		select {
		case <-finish:
			return RootScanResult{Files: []string{"example.md"}}
		case <-ctx.Done():
			return RootScanResult{Err: ctx.Err()}
		}
	}, func(event FileIndexEvent) { events <- event })
	defer index.Close()
	root := t.TempDir()
	index.Activate([]string{root})
	<-started
	snapshot := index.Snapshots([]string{root})[0]
	if snapshot.State != "scanning" || snapshot.Version != 0 || len(snapshot.Files) != 0 {
		t.Fatalf("in-progress snapshot = %+v", snapshot)
	}
	close(finish)
	select {
	case <-events:
	case <-time.After(time.Second):
		t.Fatal("scan completion was not published")
	}
	snapshot = index.Snapshots([]string{root})[0]
	if snapshot.State != "ready" || snapshot.Version != 1 || len(snapshot.Files) != 1 {
		t.Fatalf("completed snapshot = %+v", snapshot)
	}
}

func nextIndexEvent(t *testing.T, events <-chan FileIndexEvent) FileIndexEvent {
	t.Helper()
	select {
	case event := <-events:
		return event
	case <-time.After(3 * time.Second):
		t.Fatal("index did not publish a state change")
		return FileIndexEvent{}
	}
}

func TestFileIndexCancelledRefreshKeepsCachedSnapshot(t *testing.T) {
	events := make(chan FileIndexEvent, 8)
	started := make(chan struct{}, 1)
	var scans atomic.Int32
	index := NewFileIndex(context.Background(), 4, 1, func(ctx context.Context, root string) RootScanResult {
		if scans.Add(1) == 1 {
			return RootScanResult{Files: []string{"published.md"}}
		}
		started <- struct{}{}
		<-ctx.Done()
		return RootScanResult{Files: []string{"partial.md"}}
	}, func(event FileIndexEvent) { events <- event })
	defer index.Close()
	root := t.TempDir()
	index.Activate([]string{root})
	nextIndexEvent(t, events)
	index.Activate(nil)
	index.Activate([]string{root})
	<-started
	snapshot := index.Snapshots([]string{root})[0]
	if snapshot.State != "refreshing" || snapshot.Files[0] != "published.md" {
		t.Fatalf("reactivation snapshot = %+v", snapshot)
	}
	index.Activate(nil)
	nextIndexEvent(t, events)
	snapshot = index.Snapshots([]string{root})[0]
	if snapshot.Version != 1 || snapshot.Files[0] != "published.md" || !snapshot.Stale {
		t.Fatalf("cancelled refresh replaced the cache: %+v", snapshot)
	}
}

func TestFileIndexStartupScanSurvivesDeactivation(t *testing.T) {
	events := make(chan FileIndexEvent, 8)
	started := make(chan context.Context, 1)
	finish := make(chan struct{})
	index := NewFileIndex(context.Background(), 4, 1, func(ctx context.Context, root string) RootScanResult {
		started <- ctx
		select {
		case <-finish:
			return RootScanResult{Files: []string{"prewarmed.md"}}
		case <-ctx.Done():
			return RootScanResult{Err: ctx.Err()}
		}
	}, func(event FileIndexEvent) { events <- event })
	defer index.Close()
	root := t.TempDir()
	index.Prewarm([]string{root, root})
	ctx := <-started
	index.Activate([]string{root})
	index.Activate(nil)
	if ctx.Err() != nil {
		t.Fatal("startup scan was cancelled by workspace deactivation")
	}
	close(finish)
	nextIndexEvent(t, events)
	snapshot := index.Snapshots([]string{root})[0]
	if snapshot.Version != 1 || snapshot.Files[0] != "prewarmed.md" || !snapshot.Stale {
		t.Fatalf("inactive startup snapshot = %+v", snapshot)
	}
}

func TestFileIndexScanErrorRetainsLastPublishedFiles(t *testing.T) {
	events := make(chan FileIndexEvent, 8)
	var scans atomic.Int32
	index := NewFileIndex(context.Background(), 4, 1, func(ctx context.Context, root string) RootScanResult {
		if scans.Add(1) == 1 {
			return RootScanResult{Files: []string{"complete.md"}, Truncated: true}
		}
		return RootScanResult{Files: []string{"partial.md"}, Err: errors.New("scan failed")}
	}, func(event FileIndexEvent) { events <- event })
	defer index.Close()
	root := t.TempDir()
	index.Activate([]string{root})
	nextIndexEvent(t, events)
	if index.Snapshots([]string{root})[0].State != "truncated" {
		t.Fatal("file cap was not exposed as truncation")
	}
	index.Refresh([]string{root})
	nextIndexEvent(t, events)
	snapshot := index.Snapshots([]string{root})[0]
	if snapshot.State != "error" || snapshot.Version != 1 || snapshot.Files[0] != "complete.md" {
		t.Fatalf("failed scan replaced the cache: %+v", snapshot)
	}
}

func TestFileIndexEvictsInactiveSnapshotsOnly(t *testing.T) {
	events := make(chan FileIndexEvent, 8)
	index := NewFileIndex(context.Background(), 1, 1, func(ctx context.Context, root string) RootScanResult {
		return RootScanResult{Files: []string{"file.md"}}
	}, func(event FileIndexEvent) { events <- event })
	defer index.Close()
	first, second := t.TempDir(), t.TempDir()
	index.Activate([]string{first})
	nextIndexEvent(t, events)
	index.Activate([]string{second})
	nextIndexEvent(t, events)
	if len(index.Snapshots([]string{first})) != 0 || len(index.Snapshots([]string{second})) != 1 {
		t.Fatal("LRU did not retain the active root and evict the inactive one")
	}
}

func TestFileIndexBoundsConcurrentScans(t *testing.T) {
	started := make(chan struct{}, 3)
	finish := make(chan struct{})
	events := make(chan FileIndexEvent, 8)
	var running, maximum atomic.Int32
	index := NewFileIndex(context.Background(), 4, 2, func(ctx context.Context, root string) RootScanResult {
		count := running.Add(1)
		defer running.Add(-1)
		for current := maximum.Load(); count > current && !maximum.CompareAndSwap(current, count); current = maximum.Load() {
		}
		started <- struct{}{}
		select {
		case <-finish:
			return RootScanResult{}
		case <-ctx.Done():
			return RootScanResult{Err: ctx.Err()}
		}
	}, func(event FileIndexEvent) { events <- event })
	defer index.Close()
	index.Prewarm([]string{t.TempDir(), t.TempDir(), t.TempDir()})
	<-started
	<-started
	close(finish)
	for range 3 {
		nextIndexEvent(t, events)
	}
	if maximum.Load() > 2 {
		t.Fatalf("concurrent scans = %d, want at most 2", maximum.Load())
	}
}

func TestFileIndexReactivationWaitsForCancelledWorker(t *testing.T) {
	requests := make(chan chan struct{}, 2)
	events := make(chan FileIndexEvent, 4)
	quit := make(chan struct{})
	index := NewFileIndex(context.Background(), 4, 2, func(ctx context.Context, root string) RootScanResult {
		finish := make(chan struct{})
		requests <- finish
		select {
		case <-finish:
			return RootScanResult{Files: []string{"finished.md"}}
		case <-quit:
			return RootScanResult{Err: context.Canceled}
		}
	}, func(event FileIndexEvent) { events <- event })
	defer func() {
		close(quit)
		index.Close()
	}()
	root := t.TempDir()
	index.Activate([]string{root})
	first := <-requests
	index.Activate(nil)
	index.Activate([]string{root})
	select {
	case <-requests:
		t.Fatal("reactivation overlapped the cancelled worker")
	default:
	}
	close(first)
	second := <-requests
	if snapshot := index.Snapshots([]string{root})[0]; snapshot.Version != 0 || len(snapshot.Files) != 0 {
		t.Fatalf("cancelled worker published files: %+v", snapshot)
	}
	close(second)
	nextIndexEvent(t, events)
	nextIndexEvent(t, events)
	if snapshot := index.Snapshots([]string{root})[0]; snapshot.Version != 1 {
		t.Fatalf("fresh worker did not publish exactly once: %+v", snapshot)
	}
}

func TestScanIndexRootReportsFilesystemEntryLimit(t *testing.T) {
	root := t.TempDir()
	for number := range MAX_FILES {
		name := filepath.Join(root, fmt.Sprintf("%05d.txt", number))
		if err := os.WriteFile(name, nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	result := ScanIndexRoot(context.Background(), root)
	if result.Err != nil || !result.Truncated || len(result.Files) != MAX_FILES-1 {
		t.Fatalf("capped scan = %d files, truncated %v, error %v", len(result.Files), result.Truncated, result.Err)
	}
}

func awaitIndexFiles(t *testing.T, index *FileIndex, root string, events <-chan FileIndexEvent, want []string) {
	t.Helper()
	deadline := time.NewTimer(8 * time.Second)
	defer deadline.Stop()
	for {
		snapshot := index.Snapshots([]string{root})[0]
		if snapshot.State == "ready" && equalIndexFiles(snapshot.Files, want) {
			return
		}
		select {
		case <-events:
		case <-deadline.C:
			t.Fatalf("files = %+v, want %v", snapshot, want)
		}
	}
}

func TestFileIndexNativeWatcherUpdatesMembershipAndStopsInactive(t *testing.T) {
	root := t.TempDir()
	events := make(chan FileIndexEvent, 128)
	index := NewFileIndex(context.Background(), 4, 2, ScanIndexRoot, func(event FileIndexEvent) { events <- event })
	index.EnableWatching()
	defer index.Close()
	index.Activate([]string{root})
	awaitIndexFiles(t, index, root, events, []string{})
	first := filepath.Join(root, "first.md")
	second := filepath.Join(root, "second.md")
	if err := os.WriteFile(first, nil, 0600); err != nil {
		t.Fatal(err)
	}
	awaitIndexFiles(t, index, root, events, []string{"first.md"})
	if err := os.Rename(first, second); err != nil {
		t.Fatal(err)
	}
	awaitIndexFiles(t, index, root, events, []string{"second.md"})
	if err := os.MkdirAll(filepath.Join(root, "nested"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "nested", "note.md"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	awaitIndexFiles(t, index, root, events, []string{filepath.Join("nested", "note.md"), "second.md"})
	if err := os.RemoveAll(filepath.Join(root, "nested")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(second); err != nil {
		t.Fatal(err)
	}
	awaitIndexFiles(t, index, root, events, []string{})
	index.Activate(nil)
	index.mu.Lock()
	if index.entries[indexRoot(root)].watchCancel != nil {
		t.Error("inactive root retained its watcher")
	}
	index.mu.Unlock()
}

func TestFileIndexOverflowTriggersRecoveryScan(t *testing.T) {
	root := t.TempDir()
	events := make(chan FileIndexEvent, 16)
	changes := make(chan dirwatch.Change, 1)
	var scans atomic.Int32
	index := NewFileIndex(context.Background(), 4, 1, func(ctx context.Context, path string) RootScanResult {
		scans.Add(1)
		return ScanIndexRoot(ctx, path)
	}, func(event FileIndexEvent) { events <- event })
	index.watch = func(ctx context.Context, root string, ignored func(string) bool, notify func(dirwatch.Change), ready func(error)) {
		ready(nil)
		for {
			select {
			case <-ctx.Done():
				return
			case change := <-changes:
				notify(change)
			}
		}
	}
	defer index.Close()
	index.Activate([]string{root})
	awaitIndexFiles(t, index, root, events, []string{})
	if err := os.WriteFile(filepath.Join(root, "recovered.md"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	changes <- dirwatch.Change{Path: root, Rescan: true}
	awaitIndexFiles(t, index, root, events, []string{"recovered.md"})
	if scans.Load() != 2 {
		t.Fatalf("recovery scans = %d, want 2", scans.Load())
	}
}

func TestReconcileIndexPathsIgnoresSystemFiles(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, ".DS_Store")
	if err := os.WriteFile(path, nil, 0600); err != nil {
		t.Fatal(err)
	}
	result := reconcileIndexPaths(context.Background(), root, nil, []string{path}, 1)
	if result.Err != nil || len(result.Files) != 0 {
		t.Fatalf("system file indexed: %+v", result)
	}
}

func TestFileIndexNativeWatcherFindsNewlyCreatedRoot(t *testing.T) {
	root := filepath.Join(t.TempDir(), "missing", "root")
	events := make(chan FileIndexEvent, 128)
	index := NewFileIndex(context.Background(), 4, 1, ScanIndexRoot, func(event FileIndexEvent) { events <- event })
	index.EnableWatching()
	defer index.Close()
	index.Activate([]string{root})
	awaitIndexFiles(t, index, root, events, []string{})
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "created.md"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	awaitIndexFiles(t, index, root, events, []string{"created.md"})
}

func TestFileIndexWatchEventsDuringScanRequireReconciliation(t *testing.T) {
	root := t.TempDir()
	events := make(chan FileIndexEvent, 16)
	finish := make(chan struct{})
	started := make(chan struct{}, 1)
	var scans atomic.Int32
	index := NewFileIndex(context.Background(), 4, 1, func(ctx context.Context, path string) RootScanResult {
		if scans.Add(1) == 1 {
			started <- struct{}{}
			select {
			case <-finish:
				return RootScanResult{}
			case <-ctx.Done():
				return RootScanResult{Err: ctx.Err()}
			}
		}
		return ScanIndexRoot(ctx, path)
	}, func(event FileIndexEvent) { events <- event })
	defer index.Close()
	index.Activate([]string{root})
	<-started
	path := filepath.Join(root, "during-scan.md")
	if err := os.WriteFile(path, nil, 0600); err != nil {
		t.Fatal(err)
	}
	index.NotifyPaths(path)
	close(finish)
	awaitIndexFiles(t, index, root, events, []string{"during-scan.md"})
	if scans.Load() != 2 {
		t.Fatalf("scans = %d, want initial and reconciliation", scans.Load())
	}
}

func TestFileIndexWriteThroughHintsCoalesceWithoutFullWalk(t *testing.T) {
	root := t.TempDir()
	events := make(chan FileIndexEvent, 16)
	var scans atomic.Int32
	index := NewFileIndex(context.Background(), 4, 1, func(ctx context.Context, path string) RootScanResult {
		scans.Add(1)
		return ScanIndexRoot(ctx, path)
	}, func(event FileIndexEvent) { events <- event })
	defer index.Close()
	unsubscribe := SubscribeFileChanges(index.NotifyPaths)
	defer unsubscribe()
	index.Activate([]string{root})
	awaitIndexFiles(t, index, root, events, []string{})
	path := filepath.Join(root, "hint.md")
	if err := os.WriteFile(path, nil, 0600); err != nil {
		t.Fatal(err)
	}
	for range 10 {
		NotifyFileChanges(path)
	}
	awaitIndexFiles(t, index, root, events, []string{"hint.md"})
	if scans.Load() != 1 {
		t.Fatalf("write-through hint caused %d full walks", scans.Load())
	}
}

func TestFileIndexNativeWatcherHonoursExcludedDirectories(t *testing.T) {
	original := config.Config.Notes.ExcludeDirectories
	excluded := make(map[string]bool, len(original)+1)
	for name, enabled := range original {
		excluded[name] = enabled
	}
	excluded["ignored"] = true
	config.Config.Notes.ExcludeDirectories = excluded
	defer func() { config.Config.Notes.ExcludeDirectories = original }()
	root := t.TempDir()
	events := make(chan FileIndexEvent, 128)
	index := NewFileIndex(context.Background(), 4, 1, ScanIndexRoot, func(event FileIndexEvent) { events <- event })
	index.EnableWatching()
	defer index.Close()
	index.Activate([]string{root})
	awaitIndexFiles(t, index, root, events, []string{})
	if err := os.Mkdir(filepath.Join(root, "ignored"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "ignored", "hidden.md"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "visible.md"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	awaitIndexFiles(t, index, root, events, []string{"visible.md"})
}
