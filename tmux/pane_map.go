package tmux

import (
	"iter"
	"sync"
)

type paneMap struct {
	_map   map[string]*PaneT
	_mutex sync.RWMutex
}

func newPaneMap() paneMap {
	return paneMap{_map: make(map[string]*PaneT)}
}

func (pm *paneMap) Get(key string) *PaneT {
	pm._mutex.RLock()
	pane, ok := pm._map[key]
	pm._mutex.RUnlock()

	if !ok { // todo, this probably isn't needed
		return nil
	}

	return pane
}

func (pm *paneMap) Set(key string, pane *PaneT) {
	pm._mutex.Lock()
	pm._map[key] = pane
	pm._mutex.Unlock()
}

func (pm *paneMap) Delete(key string) {
	pm._mutex.Lock()
	delete(pm._map, key)
	pm._mutex.Unlock()
}

func (pm *paneMap) Each() iter.Seq[*PaneT] {
	return func(yield func(*PaneT) bool) {

		pm._mutex.RLock()
		defer pm._mutex.RUnlock()
		for _, pane := range pm._map {
			pm._mutex.RUnlock()

			ok := yield(pane)
			pm._mutex.RLock()
			if !ok {
				break
			}
		}
	}
}
