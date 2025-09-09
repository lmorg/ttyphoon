package markdown

import "fmt"

type OutputFormatT int

const (
	OUTPUT_FORMAT_MARKDOWN OutputFormatT = iota
)

func (n *nodeT) print(fmt OutputFormatT) []rune {
	return printer[fmt][n.nodeType](n)
}

type printFnT func(*nodeT, []rune) []rune

var printer = map[OutputFormatT]map[nodeTypeT]printFnT{
	OUTPUT_FORMAT_MARKDOWN: printMarkdown,
}

var printMarkdown = map[nodeTypeT]printFnT{
	_NODE_PARAGRAPH:   func(n *nodeT, r []rune) []rune { return r },
	_NODE_LINE_SPACE:  func(n *nodeT, r []rune) []rune { return "\n\n" },
	_NODE_CODE_INLINE: func(n *nodeT, r []rune) []rune { return fmt.Sprintf("`%s`", r) },
	_NODE_CODE_BLOCK:  func(n *nodeT, r []rune) []rune { return fmt.Sprintf("\n```%s```\n", r) },
	_NODE_QUOTE:       func(n *nodeT, r []rune) []rune { return fmt.Sprintf("\n> %s\n", r) },
	_NODE_HEADING_1:   func(n *nodeT, r []rune) []rune { return fmt.Sprintf("\n# %s\n", r) },
	_NODE_HEADING_2:   func(n *nodeT, r []rune) []rune { return fmt.Sprintf("\n## %s\n", r) },
	_NODE_HEADING_3:   func(n *nodeT, r []rune) []rune { return fmt.Sprintf("\n### %s\n", r) },
	_NODE_HEADING_4:   func(n *nodeT, r []rune) []rune { return fmt.Sprintf("\n#### %s\n", r) },
	_NODE_HEADING_5:   func(n *nodeT, r []rune) []rune { return fmt.Sprintf("\n##### %s\n", r) },
	_NODE_HEADING_6:   func(n *nodeT, r []rune) []rune { return fmt.Sprintf("\n###### %s\n", r) },
}
