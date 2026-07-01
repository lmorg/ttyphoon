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

/*
const (
	zwj  rune = 0x200D
	vs16 rune = 0xFE0F
)

func _RuneWidth(r rune) int {
	if isZeroWidthRune(r) {
		return 0
	}

	if isWideRune(r) {
		return 2
	}

	return 1
}

func _StringWidth(s string) int {
	if s == "" {
		return 0
	}

	runes := []rune(s)
	w := 0
	for i := 0; i < len(runes); {
		next, cw := nextCluster(runes, i)
		w += cw
		i = next
	}

	return w
}

func _Truncate(s string, crop int, terminator string) string {
	if crop <= 0 {
		return ""
	}

	if StringWidth(s) <= crop {
		return s
	}

	tw := StringWidth(terminator)
	if tw >= crop {
		return truncateToWidth(terminator, crop)
	}

	limit := crop - tw
	b := strings.Builder{}
	runes := []rune(s)
	w := 0
	for i := 0; i < len(runes); {
		next, cw := nextCluster(runes, i)
		if w+cw > limit {
			break
		}
		b.WriteString(string(runes[i:next]))
		w += cw
		i = next
	}

	b.WriteString(terminator)
	return b.String()
}

func truncateToWidth(s string, crop int) string {
	if crop <= 0 || s == "" {
		return ""
	}

	runes := []rune(s)
	b := strings.Builder{}
	w := 0
	for i := 0; i < len(runes); {
		next, cw := nextCluster(runes, i)
		if w+cw > crop {
			break
		}
		b.WriteString(string(runes[i:next]))
		w += cw
		i = next
	}

	return b.String()
}

func nextCluster(runes []rune, i int) (int, int) {
	r := runes[i]

	if isRegionalIndicator(r) && i+1 < len(runes) && isRegionalIndicator(runes[i+1]) {
		return i + 2, 2
	}

	if isKeycapStarter(r) {
		j := i + 1
		if j < len(runes) && runes[j] == vs16 {
			j++
		}
		if j < len(runes) && runes[j] == 0x20E3 {
			return j + 1, 2
		}
	}

	if isEmojiBase(r) {
		j := i + 1
		for j < len(runes) && isEmojiInlineExtender(runes[j]) {
			j++
		}

		for j < len(runes)-1 && runes[j] == zwj {
			j++
			if j < len(runes) && isEmojiBase(runes[j]) {
				j++
				for j < len(runes) && isEmojiInlineExtender(runes[j]) {
					j++
				}
			} else {
				break
			}
		}

		return j, 2
	}

	j := i + 1
	for j < len(runes) && isZeroWidthRune(runes[j]) {
		j++
	}

	return j, RuneWidth(r)
}

func isZeroWidthRune(r rune) bool {
	if r == 0 {
		return true
	}

	if r < 0x20 || (r >= 0x7F && r < 0xA0) {
		return true
	}

	if unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Me, r) || unicode.Is(unicode.Cf, r) {
		return true
	}

	if (r >= 0xFE00 && r <= 0xFE0F) || (r >= 0xE0100 && r <= 0xE01EF) {
		return true
	}

	if r >= 0xE0020 && r <= 0xE007F {
		return true
	}

	if isEmojiModifier(r) {
		return true
	}

	return false
}

func isWideRune(r rune) bool {
	k := width.LookupRune(r).Kind()
	if k == width.EastAsianWide || k == width.EastAsianFullwidth {
		return true
	}

	return isEmojiBase(r)
}

func isEmojiBase(r rune) bool {
	return (r >= 0x1F300 && r <= 0x1FAFF) ||
		(r >= 0x1FC00 && r <= 0x1FFFD) ||
		(r >= 0x2600 && r <= 0x26FF) ||
		(r >= 0x2700 && r <= 0x27BF)
}

func isEmojiModifier(r rune) bool {
	return r >= 0x1F3FB && r <= 0x1F3FF
}

func isEmojiInlineExtender(r rune) bool {
	if r == zwj {
		return false
	}

	return r == vs16 || isEmojiModifier(r) || isZeroWidthRune(r)
}

func isRegionalIndicator(r rune) bool {
	return r >= 0x1F1E6 && r <= 0x1F1FF
}

func isKeycapStarter(r rune) bool {
	return (r >= '0' && r <= '9') || r == '#' || r == '*'
}
*/
