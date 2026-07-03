package recentlist

import "slices"

// Promote moves item to the start of the list, removing any duplicate entry.
// When limit is greater than zero, the returned list is capped to that size.
func Promote(items []string, item string, limit int) []string {
	if item == "" {
		if limit > 0 && len(items) > limit {
			return append([]string(nil), items[:limit]...)
		}
		return append([]string(nil), items...)
	}

	out := append([]string(nil), items...)
	if i := slices.Index(out, item); i >= 0 {
		out = append(out[:i], out[i+1:]...)
	}

	out = append([]string{item}, out...)
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}

	return out
}
