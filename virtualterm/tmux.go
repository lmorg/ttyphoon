package virtualterm

import "github.com/lmorg/ttyphoon/codes"

func (term *Term) tmuxRenameWindow() {
	var title []rune
	for {
		r, err := term.Pty.Read()
		if err != nil {
			return
		}

	validate:
		if r == codes.AsciiEscape {
			r, err = term.Pty.Read()
			if err != nil {
				return
			}
			if r == '\\' {
				break
			}
			goto validate
		}
		title = append(title, r)
	}

	term.renderer.SetWindowTitle(string(title))
}
