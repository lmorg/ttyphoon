package virtualterm

import (
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"

	"github.com/lmorg/ttyphoon/types"
)

var (
	_CODE_OSC = []byte{27, ']'}
)

func (term *Term) osc4ColorNumber(params []string, terminator string) {
	for i := 0; i < len(params); i += 2 {
		if params[i+1] != "?" {
			// TODO: color change not yet supported
			log.Println("color change is unsupported")
			continue
		}

		key, err := strconv.Atoi(params[i])
		if err != nil {
			// TODO: color names not yet supported
			log.Printf("invalid color key (not an integer): %s", params[i])
			continue
		}

		var c *types.Colour
		switch int32(key) {
		case -1:
			// foreground
			c = types.SGR_COLOR_FOREGROUND

		case -2:
			// background
			c = types.SGR_COLOR_BACKGROUND

		default:
			var ok bool
			c, ok = types.SGR_COLOR_256[int32(key)]
			if !ok {
				log.Printf("invalid SGR_COLOR_256 key: %d", key)
				continue
			}
		}

		// Example output:
		// ESC ] 4 ; 2 ; rgb:00ff/00ff/00ff BEL
		b := fmt.Appendf(_CODE_OSC, "4;%d;rgb:%x/%x/%x%s", key, c.Red, c.Green, c.Blue, terminator)
		term.Reply(b)
	}
}

func (term *Term) osc1xColorFgBG(key int, params []string, terminator string) {
	if len(params) != 1 {
		// Expecting:
		// ESC ] 11 ; ? BEL
		return
	}

	if params[0] != "?" {
		// TODO: color change not yet supported
		log.Println("color change is unsupported")
		return
	}

	var c *types.Colour
	switch key {
	case 10:
		c = types.SGR_COLOR_FOREGROUND

	case 11:
		c = types.SGR_COLOR_BACKGROUND
	}

	// Example output:
	// ESC ] 11 ; rgb:0000/0000/0000 BEL
	b := fmt.Appendf(_CODE_OSC, "%d;rgb:%x/%x/%x%s", key, c.Red, c.Green, c.Blue, terminator)
	term.Reply(b)
}

func (term *Term) osc7UpdatePath(params []string) {
	host, pwd, ok := parseOSC7URI(params)
	if !ok {
		return
	}
	_osc7UpdatePath(term, host, pwd)
}

func parseOSC7URI(params []string) (host, pwd string, ok bool) {
	if len(params) == 0 {
		return "", "", false
	}

	uri, err := url.Parse(strings.Join(params, ";"))
	if err != nil || !strings.EqualFold(uri.Scheme, "file") || uri.Opaque != "" || uri.User != nil {
		return "", "", false
	}
	if uri.ForceQuery || uri.RawQuery != "" || uri.Fragment != "" || !strings.HasPrefix(uri.Path, "/") {
		return "", "", false
	}
	return uri.Host, uri.Path, true
}

func _osc7UpdatePath(term *Term, host, pwd string) {
	term.tile.SetPwd(pwd)

	rowSrc := types.RowSource{
		Host: host,
		Pwd:  pwd,
	}
	term._rowSource = &rowSrc
	(*term.screen)[term.curPos().Y].Source = term._rowSource

	term.renderer.RefreshNotes()
}

func (term *Term) osc9PostNotification(params []string) {
	// OSC 9 ; [Message content goes here] ST
	term.renderer.DisplayNotification(types.NOTIFY_INFO, strings.Join(params, ";"))
}

func (term *Term) osc133UpdateState(params []string) {
	if len(params) == 0 {
		return
	}

	var state types.TermStateT
	switch params[0] {
	case "A":
		state = types.TermStatePromptStart
	case "B":
		state = types.TermStatePromptEnd
	case "C":
		state = types.TermStateCommandOutputStart
	case "D":
		state = types.TermStateCommandFinished
	default:
		return
	}
	term._termState.Store(int32(state))
}
