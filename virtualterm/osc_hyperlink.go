package virtualterm

import (
	"log"
	"strings"

	"github.com/lmorg/ttyphoon/config"
	"github.com/lmorg/ttyphoon/types"
)

func (term *Term) osc8Hyperlink(sequence string) {
	// A new opener always replaces the previous target, even if it is invalid.
	term._activeTextElement = nil
	if !config.Config.Terminal.OSC8Hyperlinks {
		return
	}
	parts := strings.SplitN(sequence, ";", 3)
	if len(parts) != 3 {
		log.Printf("WARNING: Invalid OSC 8 hyperlink: missing parameters or URI delimiter")
		return
	}
	if parts[2] == "" {
		return
	}
	if err := term.beginTextElement(types.ELEMENT_ID_HYPERLINK, types.NewApcSliceNoParse([]string{"", parts[2]})); err != nil {
		log.Printf("WARNING: Invalid OSC 8 hyperlink: %v", err)
		return
	}
}
