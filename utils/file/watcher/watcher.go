package watcher

import (
	"crypto/sha256"
	"log"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/fsnotify/fsnotify"
)

const (
	// Hash files up to this size when suppressing self-write events.
	maxHashBytes int64 = 8 * 1024 * 1024
	// Fallback suppression window used when file content is too large to hash cheaply.
	largeFileIgnoreWindow = 750 * time.Millisecond
	// Safety expiry for hash-based expected writes.
	expectedWriteTTL = 2 * time.Second
)

type expectedWriteT struct {
	hash    [32]byte
	size    int64
	hasHash bool
	until   time.Time
}

var (
	watcher *fsnotify.Watcher
	onEvent atomic.Value

	mu             sync.Mutex
	files          = map[string]struct{}{}
	dirs           = map[string]struct{}{}
	expectedWrites = map[string]expectedWriteT{}
)

func init() {
	var err error
	watcher, err = fsnotify.NewWatcher()
	if err != nil {
		log.Println(err)
	} else {
		onEvent.Store((func(string))(nil))
		go listen()
	}
}

// CleanPath returns a file's path in the form used by change events.
func CleanPath(filename string) string {
	if abs, err := filepath.Abs(filename); err == nil {
		filename = abs
	}
	return filepath.Clean(filename)
}

// SetFiles replaces the set of watched files. Their parent directories are
// watched, so files that are replaced by a rename are still seen.
func SetFiles(filenames []string) {
	if watcher == nil {
		return
	}

	nextFiles := make(map[string]struct{}, len(filenames))
	nextDirs := make(map[string]struct{}, len(filenames))
	for _, filename := range filenames {
		if filename == "" {
			continue
		}
		clean := CleanPath(filename)
		nextFiles[clean] = struct{}{}
		nextDirs[filepath.Dir(clean)] = struct{}{}
	}

	mu.Lock()
	defer mu.Unlock()

	for dir := range dirs {
		if _, ok := nextDirs[dir]; ok {
			continue
		}
		if err := watcher.Remove(dir); err != nil {
			log.Printf("[debug] watcher: cannot unwatch %s: %v", dir, err)
		}
	}
	for dir := range nextDirs {
		if _, ok := dirs[dir]; ok {
			continue
		}
		if err := watcher.Add(dir); err != nil {
			log.Printf("[warn] watcher: cannot watch %s: %v", dir, err)
			delete(nextDirs, dir)
		}
	}
	files = nextFiles
	dirs = nextDirs
}

func isWatched(filename string) bool {
	mu.Lock()
	defer mu.Unlock()
	_, ok := files[filename]
	return ok
}

func SetEventCallback(fn func(string)) {
	onEvent.Store(fn)
}

// NoteWrite records an expected self-write so watcher events from this process
// can be suppressed. Files up to maxHashBytes use hash matching; larger files
// fall back to a short time-window suppression. Each file has its own
// expectation, so saving several files at once suppresses each one's events.
func NoteWrite(filename string, content []byte) {
	clean := CleanPath(filename)
	now := time.Now()

	mu.Lock()
	defer mu.Unlock()

	for path, expected := range expectedWrites {
		if now.After(expected.until) {
			delete(expectedWrites, path)
		}
	}

	if int64(len(content)) <= maxHashBytes {
		expectedWrites[clean] = expectedWriteT{
			hash:    sha256.Sum256(content),
			size:    int64(len(content)),
			hasHash: true,
			until:   now.Add(expectedWriteTTL),
		}
		return
	}

	// Large file fallback: keep a short suppression window to avoid expensive hashing.
	expectedWrites[clean] = expectedWriteT{until: now.Add(largeFileIgnoreWindow)}
}

// shouldIgnoreSelfWrite reports whether an event is from this process's own
// write. A write can raise several events, so a match stays until it expires.
func shouldIgnoreSelfWrite(eventPath string) bool {
	mu.Lock()
	expected, ok := expectedWrites[eventPath]
	if ok && time.Now().After(expected.until) {
		delete(expectedWrites, eventPath)
		ok = false
	}
	mu.Unlock()

	if !ok {
		return false
	}
	if !expected.hasHash {
		return true
	}

	fi, err := os.Stat(eventPath)
	if err != nil || fi.Size() != expected.size || fi.Size() > maxHashBytes {
		return false
	}

	b, err := os.ReadFile(eventPath)
	if err != nil || int64(len(b)) != expected.size {
		return false
	}

	return sha256.Sum256(b) == expected.hash
}

func listen() {
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if !event.Has(fsnotify.Write) && !event.Has(fsnotify.Create) && !event.Has(fsnotify.Rename) && !event.Has(fsnotify.Remove) {
				continue
			}

			eventPath := filepath.Clean(event.Name)
			if !isWatched(eventPath) {
				continue
			}

			if shouldIgnoreSelfWrite(eventPath) {
				continue
			}

			emitEvent(eventPath)

		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Println("error:", err)
		}
	}
}

func Close() {
	if watcher != nil {
		watcher.Close()
	}
}

func emitEvent(filename string) {
	fn, _ := onEvent.Load().(func(string))
	if fn == nil {
		return
	}

	fn(filepath.Clean(filename))
}
