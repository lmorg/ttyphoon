package shell

import (
	"log"
	"path/filepath"

	"github.com/lmorg/ttyphoon/types"
	"github.com/lmorg/ttyphoon/window/elements/element_content_editable"
)

func UpdateReadline(term types.Term, s string) {
	shell := filepath.Base(term.GetEnvVars()["SHELL"])

	if shell == "murex" {
		if term.State() == types.TermStatePromptStart || term.State() == types.TermStatePromptEnd {
			log.Print("[trace] UpdateReadline: " + s)
			element_content_editable.Reply(term, s)
			//time.Sleep(250 * time.Millisecond)
			term.Reply([]byte{'\n'})
		} else {
			log.Print("[error] Cannot update command line prompt because terminal doesn't appear to be ready to accept input")
		}
	} else {
		term.Reply([]byte(s))
	}
}
