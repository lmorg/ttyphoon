package markdown

import "fmt"

var printMarkdown = map[nodeTypeT]printFnT{
	_NODE_PARAGRAPH: func(n *nodeT, s string) string {
		p, ok := n.meta.(paragraphT)
		if !ok {
			panic(fmt.Sprintf("meta is typeOf %T in: '%s'", n.meta, string(n.text)))
		}

		switch {
		case p.Is(_PARAGRAPH_BOLD_BEGIN):
			s = "**" + s
		case p.Is(_PARAGRAPH_ITALIC_BEGIN):
			s = "*" + s
		case p.Is(_PARAGRAPH_UNDERLINE_BEGIN):
			s = "_" + s
		case p.Is(_PARAGRAPH_STRIKETHROUGH_BEGIN):
			s = "~~" + s
		}

		switch {
		case p.Is(_PARAGRAPH_BOLD_END):
			s += "**"
		case p.Is(_PARAGRAPH_ITALIC_END):
			s += "*"
		case p.Is(_PARAGRAPH_UNDERLINE_END):
			s += "_"
		case p.Is(_PARAGRAPH_STRIKETHROUGH_END):
			s += "~~"
		}

		switch {
		case p.Is(_PARAGRAPH_CONTINUOUS_LINE):
			return " " + s
		case p.Is(_PARAGRAPH_NESTED):
			return s
		default:
			return "\n" + s
		}
	},

	_NODE_LINE_SPACE:  func(n *nodeT, s string) string { return "\n\n" },
	_NODE_CODE_INLINE: func(n *nodeT, s string) string { return fmt.Sprintf("`%s`", s) },
	_NODE_CODE_BLOCK:  func(n *nodeT, s string) string { return fmt.Sprintf("```%s\n%s```", n.meta, s) },
	_NODE_QUOTE:       func(n *nodeT, s string) string { return fmt.Sprintf("> %s", s) },
	_NODE_HEADING_1:   func(n *nodeT, s string) string { return fmt.Sprintf("# %s", s) },
	_NODE_HEADING_2:   func(n *nodeT, s string) string { return fmt.Sprintf("## %s", s) },
	_NODE_HEADING_3:   func(n *nodeT, s string) string { return fmt.Sprintf("### %s", s) },
	_NODE_HEADING_4:   func(n *nodeT, s string) string { return fmt.Sprintf("#### %s", s) },
	_NODE_HEADING_5:   func(n *nodeT, s string) string { return fmt.Sprintf("##### %s", s) },
	_NODE_HEADING_6:   func(n *nodeT, s string) string { return fmt.Sprintf("###### %s", s) },
}
