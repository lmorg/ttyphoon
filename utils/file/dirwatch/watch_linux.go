//go:build linux

package dirwatch

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"github.com/fsnotify/fsnotify"
)

type linuxWatcher struct {
	native  *fsnotify.Watcher
	changes chan Change
	errors  chan error
	cancel  context.CancelFunc
	done    chan struct{}
	once    sync.Once
}

var errWatchBudget = errors.New("recursive directory watch entry limit reached")

func addDirectories(ctx context.Context, watcher *fsnotify.Watcher, path string, ignored func(string) bool) error {
	count := len(watcher.WatchList())
	return filepath.WalkDir(path, func(name string, entry fs.DirEntry, err error) error {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		count++
		if count > MaxEntries {
			return errWatchBudget
		}
		if entry.IsDir() {
			if ignored(name) {
				return filepath.SkipDir
			}
			return watcher.Add(name)
		}
		return nil
	})
}

func startNative(parent context.Context, root string, ignored func(string) bool, watchFileWrites bool) (nativeWatcher, error) {
	native, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	ancestor, err := existingAncestor(root)
	if err == nil {
		err = native.Add(ancestor)
	}
	if err == nil {
		err = addDirectories(parent, native, root, ignored)
	}
	if err != nil {
		native.Close()
		return nil, err
	}
	ctx, cancel := context.WithCancel(parent)
	watcher := &linuxWatcher{native: native, changes: make(chan Change, 128), errors: make(chan error, 1), cancel: cancel, done: make(chan struct{})}
	go func() {
		defer close(watcher.done)
		fail := func(err error) {
			select {
			case watcher.errors <- err:
			case <-ctx.Done():
			}
		}
		for {
			select {
			case <-ctx.Done():
				return
			case err, ok := <-native.Errors:
				if !ok {
					return
				}
				fail(err)
				return
			case event, ok := <-native.Events:
				if !ok {
					return
				}
				if !event.Has(fsnotify.Create) && !event.Has(fsnotify.Remove) && !event.Has(fsnotify.Rename) && !(watchFileWrites && event.Has(fsnotify.Write)) {
					continue
				}
				path := filepath.Clean(event.Name)
				relative, err := filepath.Rel(root, path)
				if !within(root, path) {
					if within(path, root) && event.Has(fsnotify.Create) {
						if err := addDirectories(ctx, native, path, func(name string) bool {
							return !within(name, root) && !within(root, name) || ignored(name) && within(root, name)
						}); err != nil {
							fail(err)
							return
						}
						select {
						case watcher.changes <- Change{Path: root, Rescan: true}:
						case <-ctx.Done():
							return
						}
					}
					continue
				}
				if err != nil || relative == ".." || filepath.IsAbs(relative) || ignored(path) {
					continue
				}
				if event.Has(fsnotify.Create) {
					if info, err := os.Stat(path); err == nil && info.IsDir() {
						if err := addDirectories(ctx, native, path, ignored); err != nil {
							fail(err)
							return
						}
					}
				}
				rescan := path == root
				select {
				case watcher.changes <- Change{Path: path, Rescan: rescan}:
				case <-ctx.Done():
					return
				}
				if path == root && (event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename)) {
					fail(errors.New("watched root moved or removed"))
					return
				}
			}
		}
	}()
	return watcher, nil
}

func (watcher *linuxWatcher) Changes() <-chan Change { return watcher.changes }
func (watcher *linuxWatcher) Errors() <-chan error   { return watcher.errors }
func (watcher *linuxWatcher) Close() {
	watcher.once.Do(func() {
		watcher.cancel()
		watcher.native.Close()
		<-watcher.done
	})
}
