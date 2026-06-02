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
	path    string
	hash    [32]byte
	size    int64
	hasHash bool
	until   time.Time
}

var (
	watcher *fsnotify.Watcher
	file    atomic.Value
	onEvent atomic.Value

	mu            sync.Mutex
	expectedWrite expectedWriteT
)

func init() {
	var err error
	watcher, err = fsnotify.NewWatcher()
	if err != nil {
		log.Println(err)
	} else {
		file.Store("") // initialize it as a string
		onEvent.Store((func(string))(nil))
		go listen()
	}
}

func Watch(filename string) {
	if watcher == nil {
		return
	}

	for _, f := range watcher.WatchList() {
		err := watcher.Remove(f)
		if err != nil {
			log.Println(err)
		}
	}

	abs, err := filepath.Abs(filename)
	if err == nil {
		filename = abs
	}

	clean := filepath.Clean(filename)
	file.Store(clean)
	err = watcher.Add(filepath.Dir(clean))
	if err != nil {
		log.Println(err)
	}
}

func SetEventCallback(fn func(string)) {
	onEvent.Store(fn)
}

// NoteWrite records an expected self-write so watcher events from this process
// can be suppressed. Files up to maxHashBytes use hash matching; larger files
// fall back to a short time-window suppression.
func NoteWrite(filename string, content []byte) {
	abs, err := filepath.Abs(filename)
	if err == nil {
		filename = abs
	}

	clean := filepath.Clean(filename)
	now := time.Now()

	mu.Lock()
	defer mu.Unlock()

	expectedWrite = expectedWriteT{path: clean}

	if int64(len(content)) <= maxHashBytes {
		expectedWrite.hash = sha256.Sum256(content)
		expectedWrite.size = int64(len(content))
		expectedWrite.hasHash = true
		expectedWrite.until = now.Add(expectedWriteTTL)
		return
	}

	// Large file fallback: keep a short suppression window to avoid expensive hashing.
	expectedWrite.hasHash = false
	expectedWrite.until = now.Add(largeFileIgnoreWindow)
}

func shouldIgnoreSelfWrite(eventPath string) bool {
	mu.Lock()
	defer mu.Unlock()

	if expectedWrite.path == "" || expectedWrite.path != eventPath {
		return false
	}

	if time.Now().After(expectedWrite.until) {
		expectedWrite = expectedWriteT{}
		return false
	}

	if !expectedWrite.hasHash {
		return true
	}

	fi, err := os.Stat(eventPath)
	if err != nil {
		return false
	}
	if fi.Size() != expectedWrite.size || fi.Size() > maxHashBytes {
		return false
	}

	b, err := os.ReadFile(eventPath)
	if err != nil {
		return false
	}
	if int64(len(b)) != expectedWrite.size {
		return false
	}

	if sha256.Sum256(b) == expectedWrite.hash {
		expectedWrite = expectedWriteT{}
		return true
	}

	return false
}

func listen() {
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if !event.Has(fsnotify.Write) && !event.Has(fsnotify.Create) && !event.Has(fsnotify.Rename) {
				continue
			}

			f, ok := file.Load().(string)
			if !ok || f == "" {
				continue
			}

			eventPath := filepath.Clean(event.Name)
			if eventPath != f {
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
