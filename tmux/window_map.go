package tmux

import (
	"iter"
	"sync"
)

type windowMap struct {
	_map   map[string]*WindowT
	_mutex sync.RWMutex
}

func newWindowMap() windowMap {
	return windowMap{_map: make(map[string]*WindowT)}
}

func (wm *windowMap) Get(key string) *WindowT {
	wm._mutex.RLock()
	window, ok := wm._map[key]
	wm._mutex.RUnlock()

	if !ok { // todo, this probably isn't needed
		return nil
	}

	return window
}

func (wm *windowMap) Set(key string, window *WindowT) {
	wm._mutex.Lock()
	wm._map[key] = window
	wm._mutex.Unlock()
}

func (wm *windowMap) Delete(key string) {
	wm._mutex.Lock()
	delete(wm._map, key)
	wm._mutex.Unlock()
}

func (wm *windowMap) Each() iter.Seq[*WindowT] {
	return func(yield func(*WindowT) bool) {

		wm._mutex.RLock()
		//defer wm._mutex.RUnlock()
		for _, window := range wm._map {
			wm._mutex.RUnlock()

			ok := yield(window)
			wm._mutex.RLock()
			if !ok {
				break
			}
		}

		wm._mutex.RUnlock()
	}
}
