package dirwatch

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const MaxEntries = 100_000

type Change struct {
	Path   string
	Rescan bool
	Poll   bool
}

type Options struct {
	WatchFileWrites bool
}

type nativeWatcher interface {
	Changes() <-chan Change
	Errors() <-chan error
	Close()
}

func Run(ctx context.Context, root string, ignored func(string) bool, notify func(Change), ready func(error)) {
	RunWithOptions(ctx, root, ignored, Options{}, notify, ready)
}

func RunWithOptions(ctx context.Context, root string, ignored func(string) bool, options Options, notify func(Change), ready func(error)) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		ready(errors.New("directory indexing supports only macOS and Linux"))
		<-ctx.Done()
		return
	}
	watcher, err := startNative(ctx, root, ignored, options.WatchFileWrites)
	ready(err)
	if err != nil {
		poll(ctx, root, notify)
		return
	}
	for {
		select {
		case <-ctx.Done():
			watcher.Close()
			return
		case change, ok := <-watcher.Changes():
			if !ok {
				watcher.Close()
				poll(ctx, root, notify)
				return
			}
			if change.Rescan || !ignored(change.Path) {
				notify(change)
			}
		case _, ok := <-watcher.Errors():
			watcher.Close()
			if ok && ctx.Err() == nil {
				notify(Change{Path: root, Rescan: true})
			}
			poll(ctx, root, notify)
			return
		}
	}
}

func poll(ctx context.Context, root string, notify func(Change)) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			notify(Change{Path: root, Rescan: true, Poll: true})
		}
	}
}

func within(root, path string) bool {
	relative, err := filepath.Rel(root, path)
	return err == nil && relative != ".." && !filepath.IsAbs(relative) && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func existingAncestor(root string) (string, error) {
	for path := filepath.Dir(root); ; path = filepath.Dir(path) {
		info, err := os.Stat(path)
		if err == nil && info.IsDir() {
			return path, nil
		}
		if parent := filepath.Dir(path); parent == path {
			return "", errors.New("no existing ancestor for watched root")
		}
	}
}
