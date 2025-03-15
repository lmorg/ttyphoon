package debug

type Mutex struct{}

func (bad *Mutex) Lock()         {}
func (bad *Mutex) TryLock() bool { return true }
func (bad *Mutex) Unlock()       {}
