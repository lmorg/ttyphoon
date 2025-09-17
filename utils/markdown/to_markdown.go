package markdown

import (
	"strconv"
	"strings"
)

// ToMarkdown renders the AST back to markdown format.
func ToMarkdown(n *Node) string {
	var sb strings.Builder
	writeNode(&sb, n, 0)
	return sb.String()
}

func writeNode(sb *strings.Builder, n *Node, depth int) {
	switch n.Type {
	case NodeDocument:
		for _, c := range n.Children {
			writeNode(sb, c, 0)
		}
	case NodeHeading:
		level := 1
		if n.Meta != nil {
			if l, ok := n.Meta.(int); ok {
				level = l
			}
		}
		sb.WriteString(strings.Repeat("#", level) + " ")
		for _, c := range n.Children {
			writeNode(sb, c, depth)
		}
		sb.WriteString("\n")
	case NodeParagraph:
		for _, c := range n.Children {
			writeNode(sb, c, depth)
		}
		sb.WriteString("\n\n")
	case NodeText:
		sb.WriteString(n.Text)
	case NodeBold:
		sb.WriteString("**")
		for _, c := range n.Children {
			writeNode(sb, c, depth)
		}
		sb.WriteString("**")
	case NodeItalic:
		sb.WriteString("*")
		for _, c := range n.Children {
			writeNode(sb, c, depth)
		}
		sb.WriteString("*")
	case NodeUnderline:
		sb.WriteString("__")
		for _, c := range n.Children {
			writeNode(sb, c, depth)
		}
		sb.WriteString("__")
	case NodeStrikethrough:
		sb.WriteString("~~")
		for _, c := range n.Children {
			writeNode(sb, c, depth)
		}
		sb.WriteString("~~")
	case NodeCodeInline:
		sb.WriteString("`")
		sb.WriteString(n.Text)
		sb.WriteString("`")
	case NodeCodeBlock:
		sb.WriteString("```")
		sb.WriteString("\n")
		sb.WriteString(n.Text)
		sb.WriteString("\n```")
		sb.WriteString("\n\n")
	case NodeQuote:
		for _, c := range n.Children {
			sb.WriteString("> ")
			writeNode(sb, c, depth)
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	case NodeListBullet, NodeListNumbered:
		for i, c := range n.Children {
			if c.Type == NodeListItem {
				if n.Type == NodeListBullet {
					sb.WriteString("- ")
				} else {
					sb.WriteString(strings.TrimSpace(strings.Repeat(" ", depth*2)))
					sb.WriteString(strings.TrimSpace(strconv.Itoa(i+1) + ". "))
				}
				writeNode(sb, c, depth+1)
				sb.WriteString("\n")
			}
		}
		sb.WriteString("\n")
	case NodeListItem:
		for _, c := range n.Children {
			writeNode(sb, c, depth)
		}
	case NodeTable:
		for _, row := range n.Children {
			writeNode(sb, row, depth)
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	case NodeTableRow:
		for i, cell := range n.Children {
			if i > 0 {
				sb.WriteString(" | ")
			}
			writeNode(sb, cell, depth)
		}
	case NodeTableCell:
		sb.WriteString(n.Text)
	case NodeLink:
		var label, url string
		for _, c := range n.Children {
			if c.Type == NodeLinkLabel {
				label = c.Text
			} else if c.Type == NodeLinkURL {
				url = c.Text
			}
		}
		sb.WriteString("[")
		sb.WriteString(label)
		sb.WriteString("](")
		sb.WriteString(url)
		sb.WriteString(")")
	case NodeTaskUnchecked:
		sb.WriteString("[ ] ")
	case NodeTaskChecked:
		sb.WriteString("[x] ")
	}
}
