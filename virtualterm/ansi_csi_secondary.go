package virtualterm

import "log"

func lookupSecondaryCsi(code []rune) {
	if len(code) == 0 {
		return
	}

	if code[len(code)-1] == 'u' {
		// Kitty keyboard protocol push/pop request. Keep legacy keyboard
		// encoding and do not advertise support until all flags can be honoured.
		return
	}

	log.Printf("[debug] term: Secondary CSI code ignored: '%s'", string(code))
}
