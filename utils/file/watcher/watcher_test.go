package watcher

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func collectEvents(t *testing.T) <-chan string {
	t.Helper()
	events := make(chan string, 64)
	SetEventCallback(func(path string) { events <- path })
	t.Cleanup(func() {
		SetEventCallback(nil)
		SetFiles(nil)
	})
	return events
}

// eventsWithin returns the distinct paths reported within the window.
func eventsWithin(events <-chan string, window time.Duration) map[string]bool {
	seen := map[string]bool{}
	deadline := time.After(window)
	for {
		select {
		case path := <-events:
			seen[path] = true
		case <-deadline:
			return seen
		}
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestSetFilesWatchesEveryFileAndSuppressesEachSelfWrite(t *testing.T) {
	events := collectEvents(t)
	dirA, dirB := t.TempDir(), t.TempDir()
	a := CleanPath(filepath.Join(dirA, "a.md"))
	b := CleanPath(filepath.Join(dirB, "b.md"))
	other := CleanPath(filepath.Join(dirA, "other.md"))
	for _, path := range []string{a, b, other} {
		writeFile(t, path, "start")
	}

	SetFiles([]string{a, b})

	writeFile(t, a, "outside a")
	writeFile(t, b, "outside b")
	writeFile(t, other, "outside other")
	seen := eventsWithin(events, 500*time.Millisecond)
	if !seen[a] || !seen[b] || seen[other] {
		t.Fatalf("external writes reported %v, want %s and %s only", seen, a, b)
	}

	// Saving both files before either event is handled suppresses both.
	NoteWrite(a, []byte("saved a"))
	NoteWrite(b, []byte("saved b"))
	writeFile(t, a, "saved a")
	writeFile(t, b, "saved b")
	if seen := eventsWithin(events, 500*time.Millisecond); len(seen) != 0 {
		t.Fatalf("self-writes reported %v", seen)
	}

	SetFiles([]string{b})
	writeFile(t, a, "outside a again")
	writeFile(t, b, "outside b again")
	seen = eventsWithin(events, 500*time.Millisecond)
	if seen[a] || !seen[b] {
		t.Fatalf("after unwatching %s reported %v", a, seen)
	}
}
