package virtualterm

import (
	"errors"
	"fmt"
	"log"

	"github.com/lmorg/mxtty/charset"
)

/*
	Reference documentation used:
	- Escape codes: https://invisible-island.net/xterm/ctlseqs/ctlseqs.html#h3-Controls-beginning-with-ESC
	- Character sets: https://en.wikipedia.org/wiki/National_Replacement_Character_Set

	Final character C for designating 94-character sets.
	In this list,
	o   0 , A  and B  were introduced in the VT100,
	o   most were introduced in the VT200 series,
	o   a few were introduced in the VT300 series, and
	o   a few more were introduced in the VT500 series.
	The VT220 character sets, together with a few others (such as
	Portuguese) are activated by the National Replacement
	Character Set (NRCS) controls.  The term "replacement" says
	that the character set is formed by replacing some of the
	characters in a set (termed the Multinational Character Set)
	with more useful ones for a given language.  The ASCII and DEC
	Supplemental character sets make up the two halves of the
	Multinational Character set, initially mapped to GL and GR.
	The valid final characters C for this control are:
*/

func (term *Term) fetchCharacterSet() (map[rune]rune, error) {
	param, err := term.Pty.Read()
	if err != nil {
		return nil, err
	}
	switch param {
	case '0':
		// C = 0  ⇒  DEC Special Character and Line Drawing Set, VT100.
		return charset.DecSpecialChar, nil

	case 'A':
		// C = A  ⇒  United Kingdom (UK), VT100.
		return charset.UnitedKingdom, nil

	case 'B':
		// C = B  ⇒  United States (USASCII), VT100.
		return nil, nil // (defaults to UTF-8)

	case 'C', '5':
		// C = C  or 5  ⇒  Finnish, VT200.
		return charset.Finnish, nil

	case 'H', '7':
		// C = H  or 7  ⇒  Swedish, VT200.
		return charset.Swedish, nil

	case 'K':
		// C = K  ⇒  German, VT200.
		return charset.German, nil

	case 'Q', '9':
		// C = Q  or 9  ⇒  French Canadian, VT200.
		return charset.FrenchCanadian, nil

	case 'R', 'f':
		// C = R  or f  ⇒  French, VT200.
		return charset.French, nil

	case 'Y':
		// C = Y  ⇒  Italian, VT200.
		return charset.Italian, nil

	case 'Z':
		// C = Z  ⇒  Spanish, VT200.
		return charset.Spanish, nil

	case '4':
		// C = 4  ⇒  Dutch, VT200.
		return charset.Dutch, nil

		/*
			Still TODO:
				C = " >  ⇒  Greek, VT500.
				C = % 2  ⇒  Turkish, VT500.
				C = % 6  ⇒  Portuguese, VT300.
				C = % =  ⇒  Hebrew, VT500.
				C = =  ⇒  Swiss, VT200.
				C = ` , E  or 6  ⇒  Norwegian/Danish, VT200.
			  The final character A  is a special case, since the same final
			  character is used by the VT300-control for the 96-character
			  British Latin-1.
			  There are a few other 94-character sets:

				C = <  ⇒  DEC Supplemental, VT200.
				C = <  ⇒  User Preferred Selection Set, VT300.
				C = >  ⇒  DEC Technical, VT300.
			  These are documented as 94-character sets (like USASCII)
			  without NRCS:
				C = " 4  ⇒  DEC Hebrew, VT500.
				C = " ?  ⇒  DEC Greek, VT500.
				C = % 0  ⇒  DEC Turkish, VT500.
				C = % 5  ⇒  DEC Supplemental Graphics, VT300.
				C = & 4  ⇒  DEC Cyrillic, VT500.
			  The VT520 reference manual lists a few more, but no
			  documentation has been found for the mappings:
				C = % 3  ⇒  SCS NRCS, VT500.
				C = & 5  ⇒  DEC Russian, VT500.
		*/

	default:
		e := fmt.Sprintf("DEBUG: Character set %s requested but does not exist", string(param))
		log.Print(e)
		return nil, errors.New(e)
	}
}
