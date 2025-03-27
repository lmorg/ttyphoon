package virtualterm

func (term *Term) osc7UpdatePath(params []string) {
	if len(params[0]) <= 7 { // "file://" {
		return
	}

	var (
		host, pwd []rune
		ptr       *[]rune = &host
	)
	for _, r := range params[0][7:] {
		if r == '/' {
			ptr = &pwd
		}
		*ptr = append(*ptr, r)
	}
	term._host = string(host)
	term._pwd = string(pwd)
}
