package cache

const (
	NS_INPUTBOXW_HISTORY = "inputboxw_history"
	NS_NOTESW_FILES      = "notesw_files"
	NS_NOTESW_RECENT     = "notesw_recent"
)

func InitCache() {
	initNamespace(NS_INPUTBOXW_HISTORY)
	initNamespace(NS_NOTESW_FILES)
	initNamespace(NS_NOTESW_RECENT)
}

func initNamespace(namespace string) {
	if configCacheDisabled {
		return
	}

	cache[namespace] = new(internalCacheT)
	cache[namespace].cache = make(map[string]*cacheItemT)
	createDb(namespace)
	disabled = false
}
