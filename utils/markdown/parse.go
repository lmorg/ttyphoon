package markdown

type nodeT struct {
	nodeType nodeTypeT
	children []*nodeT
	meta     any
	text     string // for leaf nodes
}

func (n *nodeT) addLeaf(nodeType nodeTypeT, text []rune, state *stateT) {
	n.children = append(n.children, &nodeT{
		nodeType: nodeType,
		text:     string(text),
	})
	state.textStyle = _STYLE_NONE
}

type fnParserT func(*nodeT, *stateT, []rune)

func (n *nodeT) addNode(nodeType nodeTypeT, parser fnParserT, state *stateT, line []rune) {
	node := &nodeT{nodeType: nodeType}
	n.children = append(n.children, node)
	state.textStyle = _STYLE_NONE
	parser(node, state, line)
}

type nodeTypeT int

const (
	_NODE_PARAGRAPH nodeTypeT = iota
	_NODE_LINE_SPACE
	_NODE_LINK
	_NODE_LINK_LABEL
	_NODE_LINK_URL
	//_NODE_BOLD
	//_NODE_ITALIC
	//_NODE_UNDERLINE
	//_NODE_STRIKETHROUGH
	_NODE_CODE_INLINE
	_NODE_CODE_BLOCK
	_NODE_QUOTE
	_NODE_HEADING_0 // this is just used to calculate the other headings
	_NODE_HEADING_1
	_NODE_HEADING_2
	_NODE_HEADING_3
	_NODE_HEADING_4
	_NODE_HEADING_5
	_NODE_HEADING_6
	_NODE_LIST_BULLET
	_NODE_LIST_NUMBERED
	_NODE_LIST_CHECK_TRUE
	_NODE_LIST_CHECK_FALSE
	_NODE_TABLE
	_NODE_TABLE_ROW
	_NODE_TABLE_CELL
	_NODE_IMAGE
)

const _MAX_HEADING = 6

type stateT struct {
	ch        chan []rune
	textStyle textStyleT
}

type textStyleT int

const (
	_STYLE_NONE  textStyleT = 0
	_STYLE_TOKEN textStyleT = 1 << iota
	_STYLE_BOLD
	_STYLE_ITALIC
	_STYLE_UNDERLINE
	_STYLE_STRIKETHROUGH
)

func (f textStyleT) Is(flag textStyleT) bool { return f&flag != 0 }
func (f *textStyleT) Set(flag textStyleT)    { *f |= flag }
func (f *textStyleT) Unset(flag textStyleT)  { *f &^= flag }

// Parse parses the given markdown runes and returns the root AST node.
func Parse(md []rune) *nodeT {
	root := &nodeT{nodeType: _NODE_PARAGRAPH, children: []*nodeT{}}

	state := &stateT{ch: make(chan []rune)}
	go splitLines(md, state.ch)

	parse(root, state)

	return root
}

// splitLines splits the input into lines (without \n or \r).
func splitLines(md []rune, ch chan []rune) {
	var line []rune
	for _, r := range md {
		switch r {
		default:
			line = append(line, r)
		case '\r':
			continue
		case '\t':
			line = append(line, ' ', ' ', ' ', ' ')
		case '\n':
			ch <- line
			line = []rune{}
		}
	}
	if len(line) > 0 {
		ch <- line
	}
	close(ch)
}

func parse(parentNode *nodeT, state *stateT) {
	for {
		line, ok := <-state.ch
		if !ok {
			break
		}

		parseLine(parentNode, state, line)
	}

}

// parseLine parses a single line and adds nodes to the parent AST node.
func parseLine(parentNode *nodeT, state *stateT, line []rune) {
	if len(line) == 0 {
		parentNode.addLeaf(_NODE_LINE_SPACE, nil, state)
		return
	}

	switch line[0] {
	case '#':
		parseHeading(parentNode, state, line)
		return

	case '>':
		parentNode.addNode(_NODE_QUOTE, parseLine, state, line[1:])
		return

	case '`':
		if len(line) >= 3 && line[1] == '`' && line[2] == '`' {
			parseCodeBlock(parentNode, state, line[3:])
			return
		}

	case ' ':
		if len(line) >= 4 && line[1] == ' ' && line[2] == ' ' && line[3] == ' ' {
			parentNode.addLeaf(_NODE_PARAGRAPH, line, state)
			return
		}
	}

	parentNode.addLeaf(_NODE_PARAGRAPH, line, state)

}

func parseHeading(parentNode *nodeT, state *stateT, line []rune) {
	i := 0
	for i < len(line) && line[i] == '#' {
		i++
	}

	parentNode.addNode(_NODE_HEADING_0+nodeTypeT(min(i, _MAX_HEADING)), parseParagraph, state, line[i:])
}

func parseCodeBlock(parentNode *nodeT, state *stateT, line []rune) {
	parentNode.addLeaf(_NODE_CODE_BLOCK, line, state)

	ok := true
	for ok {
		line, ok = <-state.ch

		if len(line) >= 3 &&
			line[len(line)-1] == '`' && line[len(line)-2] == '`' && line[len(line)-3] == '`' {
			line = line[:len(line)-2]
			ok = false
		}

		parentNode.addLeaf(_NODE_CODE_BLOCK, line, state)
	}
}

func parseParagraph(parentNode *nodeT, state *stateT, line []rune) {
	parentNode.meta = state.textStyle

	var (
		i int
		r rune
	)

	prev := func() rune {
		if i > 0 {
			return line[i-1]
		}
		return ' '
	}

	next := func() rune {
		if i < len(line) {
			return line[i+1]
		}
		return ' '
	}

	for i, r = range line {
		switch r {
		case '*': // bold
			if prev() != ' ' || next() != '*' {
				continue
			}

		case '_': // underline
			if prev() != ' ' || next() == ' ' {
				continue
			}

		case '`': // fixed width
			if prev() != ' ' || next() == ' ' {
				continue
			}

		case '~': // strikethrough
			if prev() != ' ' || next() == ' ' {
				continue
			}

		case '[': // link

		case '!': // image
			if next() != '[' {
				continue
			}

		default:
		}
	}

	parentNode.text = string(line)
}
