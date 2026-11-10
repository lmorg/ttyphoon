package copy_format

import (
	"bytes"
	"encoding/json"
	"fmt"
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
		case "strong":
			style.Bold = true
		case "em":
			style.Italic = true
		case "del":
			style.Strike = true
		case "code", "pre":
			style.Code = true
		case "a":
			link = attr(node, "href")
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

func slackElement(tag string, attributes ...html.Attribute) *html.Node {
	return &html.Node{Type: html.ElementNode, Data: tag, DataAtom: atom.Lookup([]byte(tag)), Attr: attributes}
}

func renderSlackHTML(root *html.Node) (string, error) {
	var clone func(*html.Node) (*html.Node, error)
	clone = func(source *html.Node) (*html.Node, error) {
		node := &html.Node{Type: source.Type, Data: source.Data, DataAtom: source.DataAtom, Attr: append([]html.Attribute(nil), source.Attr...)}
		for child := source.FirstChild; child != nil; child = child.NextSibling {
			copied, err := clone(child)
			if err != nil {
				return nil, err
			}
			node.AppendChild(copied)
		}
		if source.Type != html.ElementNode || source.Data != "table" {
			return node, nil
		}
		table := slackTable{Type: "table", Rows: [][]slackCell{}}
		for _, row := range tableRows(source) {
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
		node, err := clone(source)
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
			section.AppendChild(node)
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
