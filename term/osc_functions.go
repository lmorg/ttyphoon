package virtualterm

import "github.com/lmorg/mxtty/types"

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
	rowSrc := types.RowSource{
		Host: string(host),
		Pwd:  string(pwd),
	}
	term._rowSource = &rowSrc
	(*term.screen)[term.curPos().Y].Source = term._rowSource
}
