//go:build darwin && cgo

package dirwatch

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsevents"
)

type darwinWatcher struct {
	stream  *fsevents.EventStream
	changes chan Change
	errors  chan error
	stop    chan struct{}
	done    chan struct{}
	once    sync.Once
}

func startNative(ctx context.Context, root string, ignored func(string) bool, watchFileWrites bool) (nativeWatcher, error) {
	ancestor, err := existingAncestor(root)
	if err != nil {
		return nil, err
	}
	resolvedRoot := root
	if resolved, err := filepath.EvalSymlinks(root); err == nil {
		resolvedRoot = resolved
	} else if resolved, err := filepath.EvalSymlinks(ancestor); err == nil {
		if relative, err := filepath.Rel(ancestor, root); err == nil {
			resolvedRoot = filepath.Join(resolved, relative)
		}
	}
	resolvedAncestor, err := filepath.EvalSymlinks(ancestor)
	if err != nil {
		return nil, err
	}
	paths := []string{resolvedRoot}
	if _, err := os.Stat(root); os.IsNotExist(err) {
		paths = []string{resolvedAncestor}
	}
	watcher := &darwinWatcher{
		stream: &fsevents.EventStream{
			Paths:   paths,
			Flags:   fsevents.FileEvents | fsevents.WatchRoot | fsevents.NoDefer,
			Latency: 150 * time.Millisecond,
			Events:  make(chan []fsevents.Event, 64),
		},
		changes: make(chan Change, 128), errors: make(chan error, 1),
		stop: make(chan struct{}), done: make(chan struct{}),
	}
	if err := watcher.stream.Start(); err != nil {
		return nil, err
	}
	go func() {
		defer close(watcher.done)
		for {
			select {
			case <-watcher.stop:
				return
			case events := <-watcher.stream.Events:
				for _, event := range events {
					path := filepath.Clean("/" + event.Path)
					relative, err := filepath.Rel(resolvedRoot, path)
					inside := err == nil && within(resolvedRoot, path)
					rescan := event.Flags&(fsevents.MustScanSubDirs|fsevents.UserDropped|fsevents.KernelDropped|fsevents.RootChanged|fsevents.Mount|fsevents.Unmount) != 0
					if !inside {
						if !rescan || !within(path, resolvedRoot) && event.Flags&(fsevents.UserDropped|fsevents.KernelDropped) == 0 {
							continue
						}
						relative = "."
					}
					path = filepath.Join(root, relative)
					changeFlags := fsevents.ItemCreated | fsevents.ItemRemoved | fsevents.ItemRenamed
					if watchFileWrites {
						changeFlags |= fsevents.ItemModified
					}
					if !rescan && (ignored(path) || event.Flags&changeFlags == 0) {
						continue
					}
					select {
					case watcher.changes <- Change{Path: path, Rescan: rescan}:
					case <-watcher.stop:
						return
					case <-ctx.Done():
						return
					}
					if event.Flags&fsevents.RootChanged != 0 {
						select {
						case watcher.errors <- errors.New("watched root changed"):
						case <-ctx.Done():
							return
						case <-watcher.stop:
							return
						}
						return
					}
				}
			}
		}
	}()
	return watcher, nil
}

func (watcher *darwinWatcher) Changes() <-chan Change { return watcher.changes }
func (watcher *darwinWatcher) Errors() <-chan error   { return watcher.errors }
func (watcher *darwinWatcher) Close() {
	watcher.once.Do(func() {
		stopped := make(chan struct{})
		go func() {
			watcher.stream.Stop()
			close(stopped)
		}()
		for {
			select {
			case <-stopped:
				close(watcher.stop)
				<-watcher.done
				return
			case <-watcher.stream.Events:
			}
		}
	})
}
