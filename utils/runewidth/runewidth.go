package runewidth

import (
	rw "github.com/mattn/go-runewidth"
)

func RuneWidth(r rune) int {
	return rw.RuneWidth(r)
}

func StringWidth(s string) int {
	return rw.StringWidth(s)
}

func Truncate(s string, crop int, terminator string) string {
	return rw.Truncate(s, crop, terminator)
}
