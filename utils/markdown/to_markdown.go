package markdown

import (
	"strconv"
	"strings"
)

// isTableSeparatorRow returns true if the row is a Markdown table separator (e.g., |---|---|)
func isTableSeparatorRow(row *Node) bool {
	if row.Type != NodeTableRow {
		return false
	}
	for _, cell := range row.Children {
		txt := strings.TrimSpace(cell.Text)
		if len(txt) == 0 {
			continue
		}
		for _, r := range txt {
			if r != '-' && r != ':' {
				return false
			}
		}
	}
	return true
}

// ToMarkdown renders the AST back to markdown format.
func ToMarkdown(n *Node) string {
	var sb strings.Builder
	writeNode(&sb, n, 0)
	return sb.String()
}

func writeNode(sb *strings.Builder, n *Node, depth int) {
	switch n.Type {
	case NodeDocument:
		var prevBlock NodeType = -1
		for i, c := range n.Children {
			if i > 0 && isBlockNode(c.Type) && isBlockNode(prevBlock) {
				sb.WriteString("\n")
			}
			writeNode(sb, c, 0)
			prevBlock = c.Type
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
		sb.WriteString("\n")
	case NodeText:
		if len(n.Children) > 0 {
			for _, c := range n.Children {
				writeNode(sb, c, depth)
			}
		} else {
			sb.WriteString(n.Text)
		}
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
		sb.WriteString(strings.TrimRight(n.Text, "\n"))
		sb.WriteString("\n```")
		sb.WriteString("\n")
	case NodeQuote:
		for _, c := range n.Children {
			lines := strings.Split(strings.TrimRight(renderNodeToString(c, depth), "\n"), "\n")
			for _, l := range lines {
				sb.WriteString("> " + l + "\n")
			}
		}
	case NodeListBullet, NodeListNumbered:
		for i, c := range n.Children {
			if c.Type == NodeListItem {
				indent := strings.Repeat("  ", depth)
				if n.Type == NodeListBullet {
					sb.WriteString(indent + "- ")
				} else {
					sb.WriteString(indent + strconv.Itoa(i+1) + ". ")
				}
				writeNode(sb, c, depth+1)
				sb.WriteString("\n")
			}
		}
	case NodeListItem:
		for i, c := range n.Children {
			if i == 0 {
				writeNode(sb, c, depth)
			} else if c.Type == NodeListBullet || c.Type == NodeListNumbered {
				sb.WriteString("\n")
				writeNode(sb, c, depth)
			} else {
				// fallback: write other children inline
				writeNode(sb, c, depth)
			}
		}
	case NodeTable:
		for i, row := range n.Children {
			writeNode(sb, row, depth)
			sb.WriteString("\n")
			// After header row, print separator if next row is not already a separator
			if i == 0 && len(n.Children) > 1 {
				if !isTableSeparatorRow(n.Children[1]) {
					sb.WriteString("|" + strings.Repeat("---|", len(row.Children)) + "\n")
				}
			}
		}
	case NodeTableRow:
		sb.WriteString("|")
		for _, cell := range n.Children {
			sb.WriteString(" ")
			writeNode(sb, cell, depth)
			sb.WriteString(" |")
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

// isBlockNode returns true if the node type is a block-level element
func isBlockNode(t NodeType) bool {
	switch t {
	case NodeHeading, NodeParagraph, NodeCodeBlock, NodeQuote, NodeListBullet, NodeListNumbered, NodeTable:
		return true
	default:
		return false
	}
}

// renderNodeToString renders a node to a string (used for blockquotes)
func renderNodeToString(n *Node, depth int) string {
	var sb strings.Builder
	writeNode(&sb, n, depth)
	return sb.String()
}
