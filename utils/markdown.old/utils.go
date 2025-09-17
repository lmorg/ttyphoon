package markdown

func ltrim(r []rune) []rune {
	for i := range r {
		if r[i] == ' ' || r[i] == '\t' || r[i] == '\r' || r[i] == '\n' {
			continue
		}
		return r[i:]
	}
	return []rune{}
}

func rtrim(r []rune) []rune {
	for i := len(r) - 1; i >= 0; i-- {
		if r[i] == ' ' || r[i] == '\t' || r[i] == '\r' || r[i] == '\n' {
			continue
		}
		return r[:i]
	}
	return []rune{}
}
