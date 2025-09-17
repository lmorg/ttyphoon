package markdown

type OutputFormatT int

const (
	OUTPUT_FORMAT_MARKDOWN OutputFormatT = iota
)

type printFnT func(*nodeT, string) string

func (n *nodeT) Print(fmt OutputFormatT) string {
	/*var s string

	switch n.nodeType {
	case _NODE_PARAGRAPH, _NODE_CODE_INLINE, _NODE_CODE_BLOCK, _NODE_IMAGE:
		s = string(n.text)

	case _NODE_LINE_SPACE:
		// do nothing

	default:
		for _, child := range n.children {
			s += child.Print(fmt)
		}
	}*/

	s := string(n.text)
	for _, child := range n.children {
		s += child.Print(fmt)
	}

	printer, ok := printers[fmt][n.nodeType]
	if !ok {
		return s
	}
	return printer(n, s)
}

var printers = map[OutputFormatT]map[nodeTypeT]printFnT{
	OUTPUT_FORMAT_MARKDOWN: printMarkdown,
}
