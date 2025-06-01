package runewidth

import (
	"github.com/forPelevin/gomoji"
	mattyn_runewidth "github.com/mattn/go-runewidth"
	"github.com/rivo/uniseg"
)

func RuneWidth(r rune) int {
	if mattyn_runewidth.RuneWidth(r) == 2 ||
		uniseg.StringWidth(string(r)) == 2 ||
		gomoji.ContainsEmoji(string(r)) {
		return 2
	}

	return 1
}

func StringWidth(s string) int {
	return max(mattyn_runewidth.StringWidth(s), uniseg.StringWidth(s))
}

func Truncate(s string, crop int, terminator string) string {
	return mattyn_runewidth.Truncate(s, crop, terminator)
}
