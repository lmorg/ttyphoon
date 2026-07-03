package notes

import (
	"strings"

	"github.com/lmorg/ttyphoon/utils/cache"
	"github.com/lmorg/ttyphoon/utils/recentlist"
)

const (
	notesFindFieldMaxItems = 30
	notesFindFieldTtlDays  = 356
)

func GetFindFieldValues(fieldName string) []string {
	key := strings.TrimSpace(fieldName)
	if key == "" {
		return []string{}
	}

	values := []string{}
	if ok := cache.Read(cache.NS_NOTES_FIND_FIELDS, key, &values); !ok || len(values) == 0 {
		return []string{}
	}

	return values
}

func AddFindFieldValue(fieldName, value string) []string {
	key := strings.TrimSpace(fieldName)
	item := strings.TrimSpace(value)
	if key == "" || item == "" {
		return GetFindFieldValues(key)
	}

	values := recentlist.Promote(GetFindFieldValues(key), item, notesFindFieldMaxItems)
	cache.Write(cache.NS_NOTES_FIND_FIELDS, key, &values, cache.Days(notesFindFieldTtlDays))
	return values
}
