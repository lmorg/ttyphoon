package cache

const (
	NS_INPUTBOXW_HISTORY = "inputboxw_history"
)

func InitCache() {
	initNamespace(NS_INPUTBOXW_HISTORY)
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
