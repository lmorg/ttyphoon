package runewidth

import (
	mattyn_runewidth "github.com/mattn/go-runewidth"
	"github.com/rivo/uniseg"
)

func RuneWidth(r rune) int {
	return uniseg.StringWidth(string(r))
}

func StringWidth(s string) int {
	return uniseg.StringWidth(s)
}

func Truncate(s string, crop int, terminator string) string {
	return mattyn_runewidth.Truncate(s, crop, terminator)
}
