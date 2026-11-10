package copy_format

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	_ "image/png"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

type Formats struct {
	HTML      string `json:"html"`
	RTF       string `json:"rtf"`
	PlainText string `json:"plainText"`
	Markdown  string `json:"markdown"`
	SlackHTML string `json:"slackHTML"`
}

// Convert formats a rendered selection, not its original Markdown source.
// Images must already be inlined as PNG data URLs by the rendering surface.
func Convert(source string) (Formats, error) {
	if len(source) > 32<<20 {
		return Formats{}, fmt.Errorf("copy content exceeds 32 MiB")
	}
	nodes, err := html.ParseFragment(strings.NewReader(source), &html.Node{Type: html.ElementNode, Data: "div", DataAtom: atom.Div})
	if err != nil {
		return Formats{}, fmt.Errorf("parse copy content: %w", err)
	}
	root := &html.Node{Type: html.ElementNode, Data: "div", DataAtom: atom.Div}
	for _, node := range nodes {
		if err := clean(root, node, 0); err != nil {
			return Formats{}, err
		}
	}
	plain := renderPlain(root)
	markdown := renderText(root, true)
	rtf := renderRTF(root)
	var removeMetadata func(*html.Node)
	removeMetadata = func(node *html.Node) {
		if node.Data == "img" {
			attributes := node.Attr[:0]
			for _, attribute := range node.Attr {
				if attribute.Key != "data-copy-source" {
					attributes = append(attributes, attribute)
				}
			}
			node.Attr = attributes
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			removeMetadata(child)
		}
	}
	removeMetadata(root)
	slackHTML, err := renderSlackHTML(root)
	if err != nil {
		return Formats{}, err
	}
	var output bytes.Buffer
	for node := root.FirstChild; node != nil; node = node.NextSibling {
		if err := html.Render(&output, node); err != nil {
			return Formats{}, fmt.Errorf("render copy HTML: %w", err)
		}
	}
	return Formats{HTML: output.String(), RTF: rtf, PlainText: plain, Markdown: markdown, SlackHTML: slackHTML}, nil
}

func HTML(source string) (string, error) {
	formats, err := Convert(source)
	return formats.HTML, err
}

func RTF(source string) (string, error) {
	formats, err := Convert(source)
	return formats.RTF, err
}

func PlainText(source string) (string, error) {
	formats, err := Convert(source)
	return formats.PlainText, err
}

var space = regexp.MustCompile(`[\t\n\r\f ]+`)

func attr(node *html.Node, key string) string {
	for _, a := range node.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func safeURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil || strings.ContainsAny(raw, "\x00\r\n\t") {
		return false
	}
	switch strings.ToLower(u.Scheme) {
	case "", "http", "https", "mailto", "file":
		return true
	}
	return false
}

func pngData(source string) ([]byte, image.Config, error) {
	const prefix = "data:image/png;base64,"
	if !strings.HasPrefix(source, prefix) {
		return nil, image.Config{}, fmt.Errorf("copy image must be an inline PNG")
	}
	data, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(source, prefix))
	if err != nil {
		return nil, image.Config{}, fmt.Errorf("decode copy image: %w", err)
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || format != "png" {
		return nil, image.Config{}, fmt.Errorf("invalid PNG copy image")
	}
	if config.Width <= 0 || config.Height <= 0 || config.Width > 16384 || config.Height > 16384 {
		return nil, image.Config{}, fmt.Errorf("copy image dimensions exceed 16384 pixels")
	}
	return data, config, nil
}

func clean(parent, source *html.Node, depth int) error {
	if depth > 256 {
		return fmt.Errorf("copy content is nested too deeply")
	}
	if source.Type == html.TextNode {
		parent.AppendChild(&html.Node{Type: html.TextNode, Data: source.Data})
		return nil
	}
	if source.Type != html.ElementNode {
		return nil
	}
	tag := source.Data
	switch tag {
	case "script", "style", "iframe", "object", "embed", "button", "input":
		return nil
	case "b":
		tag = "strong"
	case "i":
		tag = "em"
	case "strike", "s":
		tag = "del"
	}
	var attributes []html.Attribute
	switch tag {
	case "p", "div", "h1", "h2", "h3", "h4", "h5", "h6", "strong", "em", "u", "del",
		"code", "pre", "blockquote", "ul", "li", "br", "hr", "table", "thead", "tbody", "tfoot", "tr", "sup", "sub":
	case "ol":
		if start, err := strconv.Atoi(attr(source, "start")); err == nil && start > 0 {
			attributes = append(attributes, html.Attribute{Key: "start", Val: strconv.Itoa(start)})
		}
	case "th", "td":
		for _, key := range []string{"colspan", "rowspan"} {
			if span, err := strconv.Atoi(attr(source, key)); err == nil && span > 1 && span <= 1000 {
				attributes = append(attributes, html.Attribute{Key: key, Val: strconv.Itoa(span)})
			}
		}
	case "a":
		if href := attr(source, "href"); safeURL(href) {
			attributes = append(attributes, html.Attribute{Key: "href", Val: href})
		}
	case "img":
		if _, _, err := pngData(attr(source, "src")); err != nil {
			return err
		}
		for _, key := range []string{"src", "alt", "data-copy-source"} {
			if value := attr(source, key); value != "" {
				attributes = append(attributes, html.Attribute{Key: key, Val: value})
			}
		}
	default:
		for child := source.FirstChild; child != nil; child = child.NextSibling {
			if err := clean(parent, child, depth+1); err != nil {
				return err
			}
		}
		return nil
	}
	node := &html.Node{Type: html.ElementNode, Data: tag, DataAtom: atom.Lookup([]byte(tag)), Attr: attributes}
	parent.AppendChild(node)
	for child := source.FirstChild; child != nil; child = child.NextSibling {
		if err := clean(node, child, depth+1); err != nil {
			return err
		}
	}
	return nil
}

func nodeText(node *html.Node, preserve bool) string {
	if node.Type == html.TextNode {
		if preserve {
			return node.Data
		}
		return space.ReplaceAllString(node.Data, " ")
	}
	if node.Data == "br" {
		return "\n"
	}
	var text strings.Builder
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		text.WriteString(nodeText(child, preserve || node.Data == "pre"))
	}
	return text.String()
}

func tableRows(table *html.Node) [][]*html.Node {
	var rows [][]*html.Node
	var visit func(*html.Node)
	visit = func(node *html.Node) {
		if node != table && node.Data == "table" {
			return
		}
		if node.Data == "tr" {
			var cells []*html.Node
			for child := node.FirstChild; child != nil; child = child.NextSibling {
				if child.Data == "td" || child.Data == "th" {
					cells = append(cells, child)
				}
			}
			if len(cells) > 0 {
				rows = append(rows, cells)
			}
			return
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(table)
	return rows
}
