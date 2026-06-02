package cache

const (
	NS_INPUTBOXW_HISTORY = "inputboxw_history"
	NS_NOTESW_FILES      = "notesw_files"
	NS_NOTESW_RECENT     = "notesw_recent"
	NS_NOTESW_HISTORY    = "notesw_history"
	NS_NOTESW_PROJECT    = "notesw_project"
	NS_NOTESW_DOCUMENT   = "notesw_document"
)

func InitCache() {
	initNamespace(NS_INPUTBOXW_HISTORY)
	initNamespace(NS_NOTESW_FILES)
	initNamespace(NS_NOTESW_RECENT)
	initNamespace(NS_NOTESW_HISTORY)
	initNamespace(NS_NOTESW_PROJECT)
	initNamespace(NS_NOTESW_DOCUMENT)
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
