package markdown

import (
	"strings"
)

type NodeType int

const (
	NodeDocument NodeType = iota
	NodeParagraph
	NodeHeading
	NodeText
	NodeBold
	NodeItalic
	NodeUnderline
	NodeStrikethrough
	NodeCodeInline
	NodeCodeBlock
	NodeQuote
	NodeListBullet
	NodeListNumbered
	NodeListItem
	NodeTable
	NodeTableRow
	NodeTableCell
	NodeLink
	NodeLinkLabel
	NodeLinkURL
	NodeTaskUnchecked
	NodeTaskChecked
	NodeHorizontalRule
)

type Node struct {
	Type     NodeType
	Children []*Node
	Text     string
	Meta     any
}

// Parse parses GitHub Flavored Markdown and returns the root AST node.
func Parse(md string) *Node {
	lines := strings.Split(md, "\n")
	root := &Node{Type: NodeDocument}
	parser := &parserState{lines: lines, pos: 0, root: root}
	parser.parseBlocks(root)
	return root
}

type parserState struct {
	lines []string
	pos   int
	root  *Node
}

func (p *parserState) nextLine() (string, bool) {
	if p.pos >= len(p.lines) {
		return "", false
	}
	line := p.lines[p.pos]
	p.pos++
	return line, true
}

func (p *parserState) peekLine() (string, bool) {
	if p.pos >= len(p.lines) {
		return "", false
	}
	return p.lines[p.pos], true
}

func (p *parserState) parseBlocks(parent *Node) {
	for {
		line, ok := p.peekLine()
		if !ok {
			break
		}
		if strings.TrimSpace(line) == "" {
			p.pos++
			continue
		}
		if heading := parseHeading(line); heading != nil {
			parent.Children = append(parent.Children, heading)
			p.pos++
			continue
		}
		if strings.HasPrefix(line, "> ") {
			parent.Children = append(parent.Children, p.parseQuote())
			continue
		}
		if strings.HasPrefix(line, "- [ ] ") || strings.HasPrefix(line, "- [x] ") {
			parent.Children = append(parent.Children, p.parseTaskList())
			continue
		}
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") || isNumberedList(line) {
			parent.Children = append(parent.Children, p.parseList())
			continue
		}
		if strings.HasPrefix(line, "|") {
			parent.Children = append(parent.Children, p.parseTable())
			continue
		}
		if strings.HasPrefix(line, "```") {
			parent.Children = append(parent.Children, p.parseCodeBlock())
			continue
		}
		// Paragraph
		parent.Children = append(parent.Children, p.parseParagraph())
	}
}

func parseHeading(line string) *Node {
	hash := 0
	for hash < len(line) && line[hash] == '#' {
		hash++
	}
	if hash > 0 && hash <= 6 && len(line) > hash && line[hash] == ' ' {
		return &Node{
			Type:     NodeHeading,
			Meta:     hash,
			Children: []*Node{parseInlines(line[hash+1:])},
		}
	}
	return nil
}

func (p *parserState) parseQuote() *Node {
	quote := &Node{Type: NodeQuote}
	for {
		line, ok := p.peekLine()
		if !ok || !strings.HasPrefix(line, "> ") {
			break
		}
		p.pos++
		quote.Children = append(quote.Children, parseInlines(line[2:]))
	}
	return quote
}

func (p *parserState) parseTaskList() *Node {
	list := &Node{Type: NodeListBullet}
	for {
		line, ok := p.peekLine()
		if !ok || !(strings.HasPrefix(line, "- [ ] ") || strings.HasPrefix(line, "- [x] ")) {
			break
		}
		p.pos++
		checked := strings.HasPrefix(line, "- [x] ")
		item := &Node{Type: NodeListItem}
		var taskType NodeType
		if checked {
			taskType = NodeTaskChecked
		} else {
			taskType = NodeTaskUnchecked
		}
		item.Children = append(item.Children, &Node{Type: taskType})
		item.Children = append(item.Children, parseInlines(line[6:]))
		list.Children = append(list.Children, item)
	}
	return list
}

func (p *parserState) parseList() *Node {
	// Determine the indentation and list type of the first item
	line, _ := p.peekLine()
	baseIndent, listType := getListIndentAndType(line)
	list := &Node{Type: listType}
	for {
		line, ok := p.peekLine()
		if !ok {
			break
		}
		indent, itemType := getListIndentAndType(line)
		if indent < baseIndent || (itemType != NodeListBullet && itemType != NodeListNumbered) {
			break
		}
		if indent > baseIndent {
			// Nested list: parse as child of previous item
			lastItem := list.Children[len(list.Children)-1]
			nested := p.parseList()
			lastItem.Children = append(lastItem.Children, nested)
			continue
		}
		// Parse this list item
		p.pos++
		item := &Node{Type: NodeListItem}
		content := extractListContent(line)
		item.Children = append(item.Children, parseInlines(content))
		list.Children = append(list.Children, item)
	}
	return list
}

// getListIndentAndType returns the indentation level (number of leading spaces) and list type for a line
func getListIndentAndType(line string) (int, NodeType) {
	indent := 0
	for indent < len(line) && line[indent] == ' ' {
		indent++
	}
	trimmed := line[indent:]
	if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
		return indent, NodeListBullet
	}
	if isNumberedList(trimmed) {
		return indent, NodeListNumbered
	}
	return indent, -1
}

// extractListContent returns the content of a list item line (removing marker and leading spaces)
func extractListContent(line string) string {
	i := 0
	for i < len(line) && line[i] == ' ' {
		i++
	}
	trimmed := line[i:]
	if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
		return trimmed[2:]
	}
	if isNumberedList(trimmed) {
		idx := strings.Index(trimmed, ". ")
		if idx >= 0 {
			return trimmed[idx+2:]
		}
	}
	return trimmed
}

func isNumberedList(line string) bool {
	if len(line) < 3 {
		return false
	}
	for i := 0; i < len(line)-2; i++ {
		if line[i] >= '0' && line[i] <= '9' && line[i+1] == '.' && line[i+2] == ' ' {
			return true
		}
	}
	return false
}

func (p *parserState) parseTable() *Node {
	table := &Node{Type: NodeTable}
	for {
		line, ok := p.peekLine()
		if !ok || !strings.HasPrefix(line, "|") {
			break
		}
		p.pos++
		row := &Node{Type: NodeTableRow}
		cells := strings.Split(line, "|")
		for _, cell := range cells[1 : len(cells)-1] {
			row.Children = append(row.Children, &Node{Type: NodeTableCell, Text: strings.TrimSpace(cell)})
		}
		table.Children = append(table.Children, row)
	}
	return table
}

func (p *parserState) parseCodeBlock() *Node {
	p.pos++ // skip opening ```
	code := &Node{Type: NodeCodeBlock}
	for {
		line, ok := p.peekLine()
		if !ok || strings.HasPrefix(line, "```") {
			p.pos++
			break
		}
		code.Text += line + "\n"
		p.pos++
	}
	return code
}

func (p *parserState) parseParagraph() *Node {
	para := &Node{Type: NodeParagraph}
	for {
		line, ok := p.peekLine()
		if !ok || strings.TrimSpace(line) == "" || parseHeading(line) != nil || strings.HasPrefix(line, "> ") || strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "* ") || isNumberedList(line) || strings.HasPrefix(line, "|") || strings.HasPrefix(line, "```") {
			break
		}
		p.pos++
		para.Children = append(para.Children, parseInlines(line))
	}
	return para
}

// parseInlines parses inline markdown (bold, italic, code, links, strikethrough, underline)
func parseInlines(s string) *Node {
	root := &Node{Type: NodeText}
	i := 0
	for i < len(s) {
		switch {
		case strings.HasPrefix(s[i:], "**"):
			end := strings.Index(s[i+2:], "**")
			if end >= 0 {
				child := &Node{Type: NodeBold}
				child.Children = append(child.Children, parseInlines(s[i+2:i+2+end]))
				root.Children = append(root.Children, child)
				i += 2 + end + 2
				continue
			}
		case strings.HasPrefix(s[i:], "*"):
			end := strings.Index(s[i+1:], "*")
			if end >= 0 {
				child := &Node{Type: NodeItalic}
				child.Children = append(child.Children, parseInlines(s[i+1:i+1+end]))
				root.Children = append(root.Children, child)
				i += 1 + end + 1
				continue
			}
		case strings.HasPrefix(s[i:], "__"):
			end := strings.Index(s[i+2:], "__")
			if end >= 0 {
				child := &Node{Type: NodeUnderline}
				child.Children = append(child.Children, parseInlines(s[i+2:i+2+end]))
				root.Children = append(root.Children, child)
				i += 2 + end + 2
				continue
			}
		case strings.HasPrefix(s[i:], "~~"):
			end := strings.Index(s[i+2:], "~~")
			if end >= 0 {
				child := &Node{Type: NodeStrikethrough}
				child.Children = append(child.Children, parseInlines(s[i+2:i+2+end]))
				root.Children = append(root.Children, child)
				i += 2 + end + 2
				continue
			}
		case strings.HasPrefix(s[i:], "`"):
			end := strings.Index(s[i+1:], "`")
			if end >= 0 {
				child := &Node{Type: NodeCodeInline, Text: s[i+1 : i+1+end]}
				root.Children = append(root.Children, child)
				i += 1 + end + 1
				continue
			}
		case strings.HasPrefix(s[i:], "["):
			end := strings.Index(s[i:], "](")
			if end >= 0 {
				close := strings.Index(s[i+end+2:], ")")
				if close >= 0 {
					label := s[i+1 : i+end]
					url := s[i+end+2 : i+end+2+close]
					child := &Node{Type: NodeLink}
					child.Children = append(child.Children, &Node{Type: NodeLinkLabel, Text: label})
					child.Children = append(child.Children, &Node{Type: NodeLinkURL, Text: url})
					root.Children = append(root.Children, child)
					i += end + 2 + close + 1
					continue
				}
			}
		}
		// Plain text
		start := i
		for i < len(s) && !strings.HasPrefix(s[i:], "**") && !strings.HasPrefix(s[i:], "*") && !strings.HasPrefix(s[i:], "__") && !strings.HasPrefix(s[i:], "~~") && !strings.HasPrefix(s[i:], "`") && !strings.HasPrefix(s[i:], "[") {
			i++
		}
		if start < i {
			root.Children = append(root.Children, &Node{Type: NodeText, Text: s[start:i]})
		}
	}
	return root
}
