package cache

const (
	NS_INPUTBOXW_HISTORY   = "inputboxw_history"
	NS_INPUTBOXW_DRAFT     = "inputboxw_draft"
	NS_NOTES_FIND_FIELDS   = "notes_find_fields"
	NS_NOTESW_FILES        = "notesw_files"
	NS_NOTESW_HISTORY      = "notesw_history"
	NS_NOTESW_PROJECT      = "notesw_project"
	NS_NOTESW_DOCUMENT     = "notesw_document"
	NS_NOTESW_COLUMN_WIDTH = "notesw_table_column_width"
	NS_NOTESW_TABLE_STATE  = "notesw_table_state"
	NS_NOTESW_LAYOUT       = "notesw_layout"
)

func InitCache() {
	initNamespace(NS_INPUTBOXW_HISTORY)
	initNamespace(NS_INPUTBOXW_DRAFT)
	initNamespace(NS_NOTES_FIND_FIELDS)
	initNamespace(NS_NOTESW_FILES)
	initNamespace(NS_NOTESW_HISTORY)
	initNamespace(NS_NOTESW_PROJECT)
	initNamespace(NS_NOTESW_DOCUMENT)
	initNamespace(NS_NOTESW_COLUMN_WIDTH)
	initNamespace(NS_NOTESW_TABLE_STATE)
	initNamespace(NS_NOTESW_LAYOUT)
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
