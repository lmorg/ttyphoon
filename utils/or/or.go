package or

func NotEmpty(s ...string) string {
	for i := range s {
		if s[i] != "" {
			return s[i]
		}
	}

	return ""
}
