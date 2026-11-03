package notes

import "sync"

var fileChanges = struct {
	sync.Mutex
	next        uint64
	subscribers map[uint64]func(...string)
}{subscribers: make(map[uint64]func(...string))}

func SubscribeFileChanges(notify func(...string)) func() {
	fileChanges.Lock()
	fileChanges.next++
	id := fileChanges.next
	fileChanges.subscribers[id] = notify
	fileChanges.Unlock()
	return func() {
		fileChanges.Lock()
		delete(fileChanges.subscribers, id)
		fileChanges.Unlock()
	}
}

func NotifyFileChanges(paths ...string) {
	fileChanges.Lock()
	subscribers := make([]func(...string), 0, len(fileChanges.subscribers))
	for _, notify := range fileChanges.subscribers {
		subscribers = append(subscribers, notify)
	}
	fileChanges.Unlock()
	for _, notify := range subscribers {
		notify(paths...)
	}
}
