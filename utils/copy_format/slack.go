package copy_format

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type slackStyle struct {
	Bold   bool `json:"bold,omitempty"`
	Italic bool `json:"italic,omitempty"`
	Strike bool `json:"strike,omitempty"`
	Code   bool `json:"code,omitempty"`
}

type slackInline struct {
	Type  string      `json:"type"`
	Text  string      `json:"text"`
	URL   string      `json:"url,omitempty"`
	Style *slackStyle `json:"style,omitempty"`
}

type slackSection struct {
	Type     string        `json:"type"`
	Elements []slackInline `json:"elements"`
}

type slackCell struct {
	Type     string         `json:"type"`
	Text     *string        `json:"text,omitempty"`
	Elements []slackSection `json:"elements,omitempty"`
}

type slackTable struct {
	Type string        `json:"type"`
	Rows [][]slackCell `json:"rows"`
}

func slackCellElements(cell *html.Node) []slackInline {
	elements := []slackInline{}
	appendText := func(text, link string, style slackStyle) {
		if text == "" {
			return
		}
		element := slackInline{Type: "text", Text: text}
		if link != "" {
			element.Type, element.URL = "link", link
		}
		if style != (slackStyle{}) {
			element.Style = &style
		}
		if len(elements) > 0 {
			last := &elements[len(elements)-1]
			sameStyle := last.Style == nil && element.Style == nil ||
				last.Style != nil && element.Style != nil && *last.Style == *element.Style
			if last.Type == element.Type && last.URL == element.URL && sameStyle {
				last.Text += text
				return
			}
		}
		elements = append(elements, element)
	}
	lineBreak := func() {
		if len(elements) > 0 && !strings.HasSuffix(elements[len(elements)-1].Text, "\n") {
			appendText("\n", "", slackStyle{})
		}
	}
	var visit func(*html.Node, slackStyle, string)
	visit = func(node *html.Node, style slackStyle, link string) {
		if node.Type == html.TextNode {
			text := node.Data
			if !style.Code {
				text = space.ReplaceAllString(text, " ")
			}
			appendText(text, link, style)
			return
		}
		switch node.Data {
		case "strong", "b":
			style.Bold = true
		case "em":
			style.Italic = true
		case "del":
			style.Strike = true
		case "code", "pre":
			style.Code = true
		case "a":
			link = attr(node, "href")
			if strings.HasPrefix(strings.TrimSpace(link), "#") {
				link = ""
			}
		case "br":
			appendText("\n", link, style)
			return
		case "img":
			appendText(attr(node, "alt"), link, style)
			return
		}
		block := isBlock(node)
		if block {
			lineBreak()
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child, style, link)
		}
		if block {
			lineBreak()
		}
	}
	for child := cell.FirstChild; child != nil; child = child.NextSibling {
		visit(child, slackStyle{Bold: cell.Data == "th"}, "")
	}
	for len(elements) > 0 {
		last := &elements[len(elements)-1]
		last.Text = strings.TrimRight(last.Text, "\n")
		if last.Text != "" {
			break
		}
		elements = elements[:len(elements)-1]
	}
	return elements
}

func makeSlackCell(cell *html.Node) slackCell {
	elements := slackCellElements(cell)
	var plain strings.Builder
	for _, element := range elements {
		if element.Type != "text" || element.Style != nil {
			return slackCell{
				Type: "rich_text",
				Elements: []slackSection{{
					Type: "rich_text_section", Elements: elements,
				}},
			}
		}
		plain.WriteString(element.Text)
	}
	text := plain.String()
	return slackCell{Type: "raw_text", Text: &text}
}

func slackCellEmpty(cell *html.Node) bool {
	for _, element := range slackCellElements(cell) {
		if strings.TrimSpace(element.Text) != "" {
			return false
		}
	}
	return true
}

func slackElement(tag string, attributes ...html.Attribute) *html.Node {
	return &html.Node{Type: html.ElementNode, Data: tag, DataAtom: atom.Lookup([]byte(tag)), Attr: attributes}
}

func appendSlackContent(parent, node *html.Node) {
	if node.Type == html.ElementNode && node.Data == "div" && strings.TrimSpace(attr(node, "class")) == "" {
		for node.FirstChild != nil {
			child := node.FirstChild
			node.RemoveChild(child)
			appendSlackContent(parent, child)
		}
		return
	}
	parent.AppendChild(node)
}

func renderSlackHTML(root *html.Node) (string, error) {
	figure := 0
	var clone func(*html.Node, bool) (*html.Node, error)
	clone = func(source *html.Node, inTable bool) (*html.Node, error) {
		inTable = inTable || source.Type == html.ElementNode && source.Data == "table"
		node := &html.Node{Type: source.Type, Data: source.Data, DataAtom: source.DataAtom, Attr: append([]html.Attribute(nil), source.Attr...)}
		if source.Type == html.ElementNode && source.Data == "img" {
			original := attr(source, "data-copy-source")
			address, err := url.Parse(original)
			if err == nil && address.Host != "" && safeURL(original) &&
				(strings.EqualFold(address.Scheme, "http") || strings.EqualFold(address.Scheme, "https")) {
				figure++
				reference := slackElement("span")
				reference.AppendChild(slackElement("br"))
				reference.AppendChild(slackElement("br"))
				reference.AppendChild(&html.Node{Type: html.TextNode, Data: fmt.Sprintf("[fig %d: %s]", figure, original)})
				reference.AppendChild(slackElement("br"))
				reference.AppendChild(slackElement("br"))
				return reference, nil
			}
			node.Attr = nil
			for _, attribute := range source.Attr {
				if attribute.Key != "data-copy-source" {
					node.Attr = append(node.Attr, attribute)
				}
			}
		}
		if inTable && source.Type == html.ElementNode && source.Data == "strong" {
			node.Data, node.DataAtom = "b", atom.B
			node.Attr = append(node.Attr, html.Attribute{Key: "data-stringify-type", Val: "bold"})
		}
		for child := source.FirstChild; child != nil; child = child.NextSibling {
			if !inTable && child.Type == html.TextNode && strings.TrimSpace(child.Data) == "" &&
				(source.Data == "ul" || source.Data == "ol" ||
					source.Data == "li" && (isBlock(child.PrevSibling) || isBlock(child.NextSibling))) {
				continue
			}
			copied, err := clone(child, inTable)
			if err != nil {
				return nil, err
			}
			appendSlackContent(node, copied)
		}
		if source.Type == html.ElementNode && len(source.Data) == 2 &&
			source.Data[0] == 'h' && source.Data[1] >= '1' && source.Data[1] <= '6' {
			node.Data, node.DataAtom = "p", atom.P
			prefix := &html.Node{Type: html.TextNode, Data: strings.Repeat("#", int(source.Data[1]-'0')) + " "}
			node.InsertBefore(prefix, node.FirstChild)
			if !inTable {
				for node.FirstChild != nil {
					node.RemoveChild(node.FirstChild)
				}
				node.AppendChild(slackElement("br"))
				node.AppendChild(slackElement("br"))
				node.AppendChild(&html.Node{Type: html.TextNode, Data: strings.Repeat("#", int(source.Data[1]-'0')) + " _" + strings.TrimSpace(nodeText(source, false)) + "_"})
				node.AppendChild(slackElement("br"))
				node.AppendChild(slackElement("br"))
			}
		}
		if !inTable && source.Type == html.ElementNode {
			switch source.Data {
			case "strong", "em", "u":
				marker := "_"
				if source.Data == "strong" {
					marker = "*"
				}
				node.Data, node.DataAtom, node.Attr = "span", atom.Span, nil
				node.InsertBefore(&html.Node{Type: html.TextNode, Data: marker}, node.FirstChild)
				node.AppendChild(&html.Node{Type: html.TextNode, Data: marker})
			case "ul", "ol":
				node.Data, node.DataAtom, node.Attr = "div", atom.Div, nil
				if source.Parent != nil && source.Parent.Data == "li" {
					node.InsertBefore(slackElement("br"), node.FirstChild)
				}
			case "li":
				marker := "- "
				if source.Parent != nil && source.Parent.Data == "ol" {
					number, _ := strconv.Atoi(attr(source.Parent, "start"))
					if number < 1 {
						number = 1
					}
					for sibling := source.PrevSibling; sibling != nil; sibling = sibling.PrevSibling {
						if sibling.Data == "li" {
							number++
						}
					}
					marker = strconv.Itoa(number) + ". "
				}
				depth := 0
				for ancestor := source.Parent; ancestor != nil; ancestor = ancestor.Parent {
					if ancestor.Data == "li" {
						depth++
					}
				}
				node.Data, node.DataAtom, node.Attr = "div", atom.Div, nil
				node.InsertBefore(&html.Node{Type: html.TextNode, Data: strings.Repeat("  ", depth) + marker}, node.FirstChild)
				for node.LastChild != nil && (node.LastChild.Data == "br" ||
					node.LastChild.Type == html.TextNode && strings.TrimSpace(node.LastChild.Data) == "") {
					node.RemoveChild(node.LastChild)
				}
				node.AppendChild(slackElement("br"))
			case "p":
				if source.Parent != nil && source.Parent.Data == "li" {
					node.Data, node.DataAtom = "span", atom.Span
					next := source.NextSibling
					for next != nil && next.Type == html.TextNode && strings.TrimSpace(next.Data) == "" {
						next = next.NextSibling
					}
					if next != nil && next.Data != "ul" && next.Data != "ol" {
						node.AppendChild(slackElement("br"))
					}
				}
			}
		}
		if source.Type == html.ElementNode && source.Data == "a" &&
			strings.HasPrefix(strings.TrimSpace(attr(source, "href")), "#") {
			node.Data, node.DataAtom = "span", atom.Span
			node.Attr = nil
		}
		if source.Type == html.ElementNode && (source.Data == "td" || source.Data == "th") && slackCellEmpty(node) {
			for node.FirstChild != nil {
				node.RemoveChild(node.FirstChild)
			}
			node.AppendChild(&html.Node{Type: html.TextNode, Data: "-"})
		}
		if source.Type != html.ElementNode || source.Data != "table" {
			return node, nil
		}
		table := slackTable{Type: "table", Rows: [][]slackCell{}}
		for _, row := range tableRows(node) {
			cells := make([]slackCell, 0, len(row))
			for _, cell := range row {
				cells = append(cells, makeSlackCell(cell))
			}
			table.Rows = append(table.Rows, cells)
		}
		metadata, err := json.Marshal(table)
		if err != nil {
			return nil, fmt.Errorf("encode Slack table: %w", err)
		}
		wrapper := slackElement("div",
			html.Attribute{Key: "class", Val: "p-block_kit_renderer__block_wrapper"},
			html.Attribute{Key: "data-block-type", Val: "table"},
			html.Attribute{Key: "data-block-source", Val: string(metadata)},
		)
		wrapper.AppendChild(node)
		return wrapper, nil
	}
	var output bytes.Buffer
	output.WriteString(`<meta charset="utf-8">`)
	var containsTable func(*html.Node) bool
	containsTable = func(node *html.Node) bool {
		if node.Type == html.ElementNode && node.Data == "table" {
			return true
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			if containsTable(child) {
				return true
			}
		}
		return false
	}
	var emit func(*html.Node) error
	emit = func(source *html.Node) error {
		if source.Type == html.TextNode && strings.TrimSpace(source.Data) == "" &&
			(isBlock(source.PrevSibling) || isBlock(source.NextSibling)) {
			return nil
		}
		// A table must be a separate Slack block, even inside a selected div.
		if source.Data != "table" && containsTable(source) {
			for child := source.FirstChild; child != nil; child = child.NextSibling {
				if err := emit(child); err != nil {
					return err
				}
			}
			return nil
		}
		node, err := clone(source, false)
		if err != nil {
			return err
		}
		if source.Data != "table" {
			wrapper := slackElement("div",
				html.Attribute{Key: "class", Val: "p-block_kit_renderer__block_wrapper"},
				html.Attribute{Key: "data-block-type", Val: "rich_text"},
			)
			block := slackElement("div", html.Attribute{Key: "class", Val: "p-rich_text_block"})
			section := slackElement("div", html.Attribute{Key: "class", Val: "p-rich_text_section"})
			appendSlackContent(section, node)
			block.AppendChild(section)
			wrapper.AppendChild(block)
			node = wrapper
		}
		if err := html.Render(&output, node); err != nil {
			return fmt.Errorf("render Slack HTML: %w", err)
		}
		return nil
	}
	for child := root.FirstChild; child != nil; child = child.NextSibling {
		if err := emit(child); err != nil {
			return "", err
		}
	}
	return output.String(), nil
}
