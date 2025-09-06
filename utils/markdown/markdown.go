package markdown

type nodeT struct {
	nodeType nodeTypeT
	children []*nodeT
	meta     any
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

func Parse(md []rune) {
	var ast nodeT

	parse(md, &ast)
}

func parse(md []rune, parentNode *nodeT) {
	start := 0
	for i, r := range md {
		switch r {
		case `\r`, `\n`:
			parseLine(md[start], ast)
		}
	}
}

func parseLine(md []rune, parentNode *nodeT) {

}
