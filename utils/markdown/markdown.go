package markdown

type nodeT struct {
	nodeType nodeTypeT
	children []*nodeT
	meta     any
	text     string // for leaf nodes
}

type nodeTypeT int

const (
	_MD_NODE_PARAGRAPH nodeTypeT = iota
	_MD_NODE_LINK
	_MD_NODE_LINK_LABEL
	_MD_NODE_LINK_URL
	_MD_NODE_BOLD
	_MD_NODE_ITALIC
	_MD_NODE_UNDERLINE
	_MD_NODE_STRIKETHROUGH
	_MD_NODE_CODE_INLINE
	_MD_NODE_CODE_BLOCK
	_MD_NODE_QUOTE
	_MD_NODE_HEADING_1
	_MD_NODE_HEADING_2
	_MD_NODE_HEADING_3
	_MD_NODE_HEADING_4
	_MD_NODE_HEADING_5
	_MD_NODE_HEADING_6
	_MD_NODE_LIST_BULLET
	_MD_NODE_LIST_NUMBERED
	_MD_NODE_TABLE
	_MD_NODE_TABLE_ROW
	_MD_NODE_TABLE_CELL
)

// Parse parses the given markdown runes and returns the root AST node.
func Parse(md []rune) *nodeT {
	lines := splitLines(md)
	root := &nodeT{nodeType: _MD_NODE_PARAGRAPH, children: []*nodeT{}}
	for _, line := range lines {
		parseLine(line, root)
	}
	return root
}

// splitLines splits the input into lines (without \n or \r).
func splitLines(md []rune) [][]rune {
	var lines [][]rune
	var line []rune
	for _, r := range md {
		if r == '\n' || r == '\r' {
			if len(line) > 0 {
				lines = append(lines, line)
				line = nil
			}
		} else {
			line = append(line, r)
		}
	}
	if len(line) > 0 {
		lines = append(lines, line)
	}
	return lines
}

// parseLine parses a single line and adds nodes to the parent AST node.
func parseLine(md []rune, parentNode *nodeT) {
	if len(md) == 0 {
		return
	}
	// Headings: #, ##, ###, etc.
	headingLevel := 0
	i := 0
	for i < len(md) && md[i] == '#' {
		headingLevel++
		i++
	}
	if headingLevel > 0 && i < len(md) && md[i] == ' ' {
		// It's a heading
		nodeType := nodeTypeT(int(_MD_NODE_HEADING_1) + headingLevel - 1)
		headingText := string(md[i+1:])
		parentNode.children = append(parentNode.children, &nodeT{
			nodeType: nodeType,
			text:     headingText,
		})
		return
	}
	// Otherwise, treat as paragraph (with inline parsing)
	para := &nodeT{nodeType: _MD_NODE_PARAGRAPH, children: []*nodeT{}}
	parseInlines(md, para)
	parentNode.children = append(parentNode.children, para)
}

// parseInlines parses inline markdown (bold, italic, code) and adds nodes to parent.
func parseInlines(md []rune, parentNode *nodeT) {
	i := 0
	for i < len(md) {
		switch {
		case i+1 < len(md) && md[i] == '*' && md[i+1] == '*':
			// Bold
			end := findInline(md, i+2, "**")
			if end >= 0 {
				child := &nodeT{nodeType: _MD_NODE_BOLD, children: []*nodeT{}}
				parseInlines(md[i+2:end], child)
				parentNode.children = append(parentNode.children, child)
				i = end + 2
				continue
			}
		case md[i] == '*':
			// Italic
			end := findInline(md, i+1, "*")
			if end >= 0 {
				child := &nodeT{nodeType: _MD_NODE_ITALIC, children: []*nodeT{}}
				parseInlines(md[i+1:end], child)
				parentNode.children = append(parentNode.children, child)
				i = end + 1
				continue
			}
		case md[i] == '`':
			// Inline code
			end := findInline(md, i+1, "`")
			if end >= 0 {
				child := &nodeT{nodeType: _MD_NODE_CODE_INLINE, text: string(md[i+1 : end])}
				parentNode.children = append(parentNode.children, child)
				i = end + 1
				continue
			}
		}
		// Plain text
		start := i
		for i < len(md) && md[i] != '*' && md[i] != '`' {
			i++
		}
		if start < i {
			parentNode.children = append(parentNode.children, &nodeT{
				nodeType: -1, // text node
				text:     string(md[start:i]),
			})
		}
	}
}

// findInline finds the end index of an inline marker (e.g., *, **, `)
func findInline(md []rune, start int, marker string) int {
	m := []rune(marker)
	for i := start; i <= len(md)-len(m); i++ {
		match := true
		for j := 0; j < len(m); j++ {
			if md[i+j] != m[j] {
				match = false
				break
			}
		}
		if match {
			return i
		}
	}
	return -1
}
