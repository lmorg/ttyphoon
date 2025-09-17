package markdown

import (
	"encoding/json"
	"fmt"
)

type nodeT struct {
	nodeType nodeTypeT
	children []*nodeT
	meta     any
	text     []rune
}

func (n *nodeT) dump() any {
	var children []any
	for i := range n.children {
		children = append(children, n.children[i].dump())
	}
	var meta any
	if n.meta != nil {
		meta = fmt.Sprintf("%T(%v)", n.meta, n.meta)
	} else {
		meta = "n/a"
	}
	return map[string]any{
		"Type":     n.nodeType.String(),
		"Meta":     meta,
		"Text":     string(n.text),
		"Children": children,
	}
}

func (n *nodeT) String() string {
	b, err := json.MarshalIndent(n.dump(), "    ", "    ")
	if err != nil {
		return err.Error()
	}
	return string(b)
}

func (n *nodeT) addLeaf(nodeType nodeTypeT, text []rune, state *stateT, meta any) {
	n._addLeaf(nodeType, text, meta)
	state.paragraph = _PARAGRAPH_NONE
}

func (n *nodeT) addParagraph(text []rune, state *stateT) {
	n._addLeaf(_NODE_PARAGRAPH, text, state.paragraph)
	state.paragraph.Set(_PARAGRAPH_CONTINUOUS_LINE)
}

func (n *nodeT) _addLeaf(nodeType nodeTypeT, text []rune, meta any) {
	n.children = append(n.children, &nodeT{
		nodeType: nodeType,
		text:     text,
		meta:     meta,
	})
}

type fnParserT func(*nodeT, *stateT, []rune)

func (n *nodeT) addNode(nodeType nodeTypeT, parser fnParserT, state *stateT, line []rune) {
	n.children = append(n.children, node)
	state.paragraph = _PARAGRAPH_NONE
	parser(node, state, line)
}

type stateT struct {
	ch        chan []rune
	paragraph paragraphT
}

type paragraphT int

const (
	_PARAGRAPH_NONE   paragraphT = 0
	_PARAGRAPH_NESTED paragraphT = 1 << (iota - 1)
	_PARAGRAPH_CONTINUOUS_LINE
	_PARAGRAPH_BOLD_BEGIN
	_PARAGRAPH_BOLD_END
	_PARAGRAPH_ITALIC_BEGIN
	_PARAGRAPH_ITALIC_END
	_PARAGRAPH_UNDERLINE_BEGIN
	_PARAGRAPH_UNDERLINE_END
	_PARAGRAPH_STRIKETHROUGH_BEGIN
	_PARAGRAPH_STRIKETHROUGH_END
)

func (f paragraphT) Is(flag paragraphT) bool { return f&flag != 0 }
func (f *paragraphT) Set(flag paragraphT)    { *f |= flag }
func (f *paragraphT) Unset(flag paragraphT)  { *f &^= flag }

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
		//parentNode.lfCount++
		//if parentNode.lfCount == 1 {
		parentNode.addLeaf(_NODE_LINE_SPACE, nil, state, nil)
		//}
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
			//parentNode.addLeaf(_NODE_PARAGRAPH, line, state, nil)
			return
		}
	}

	parseParagraph(parentNode, state, line)
}

func parseHeading(parentNode *nodeT, state *stateT, line []rune) {
	i := 0
	for i < len(line) && line[i] == '#' {
		i++
	}

	parentNode.addNode(_NODE_HEADING_0+nodeTypeT(min(i, _MAX_HEADING)), parseParagraph, state, ltrim(line[i:]))
}

func parseCodeBlock(parentNode *nodeT, state *stateT, line []rune) {
	meta := string(line)
	var r []rune

	ok := true
	for ok {
		line, ok = <-state.ch

		if len(line) >= 3 &&
			line[len(line)-1] == '`' && line[len(line)-2] == '`' && line[len(line)-3] == '`' {
			line = line[:len(line)-3]
			ok = false
		}

		if !ok && len(line) == 0 {
			break
		}

		r = append(r, line...)
		r = append(r, '\n')
	}

	parentNode.addLeaf(_NODE_CODE_BLOCK, r, state, meta)
}

func parseParagraph(parentNode *nodeT, state *stateT, line []rune) {
	//parentNode.meta = state.paragraph

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
		if i+1 < len(line) {
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
			panic("bold")

		case '_': // underline
			if prev() != ' ' || next() == ' ' {
				continue
			}
			panic("underline")

		case '`': // fixed width
			if prev() != ' ' || next() == ' ' {
				continue
			}
			panic("fixed width")

		case '~': // strikethrough
			if prev() != ' ' || next() == ' ' {
				continue
			}
			panic("strikethrough")

		case '[': // link
			panic("link")

		case '!': // image
			if next() != '[' {
				continue
			}
			panic("image")

		default:
		}
	}

	parentNode.addParagraph(line, state)
}
