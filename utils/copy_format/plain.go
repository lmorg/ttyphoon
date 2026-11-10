package copy_format

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/net/html"
)

func isBlock(node *html.Node) bool {
	if node == nil || node.Type != html.ElementNode {
		return false
	}
	switch node.Data {
	case "p", "div", "h1", "h2", "h3", "h4", "h5", "h6", "pre", "blockquote", "ul", "ol", "table", "hr":
		return true
	}
	return false
}

func renderPlain(root *html.Node) string {
	return renderText(root, false)
}

var markdownLiteral = strings.NewReplacer(
	`\`, `\\`, "`", "\\`", "*", `\*`, "_", `\_`, "[", `\[`, "]", `\]`,
	"<", `\<`, ">", `\>`, "~", `\~`, "&", "&amp;",
	"-", `\-`, "+", `\+`, "#", `\#`, "=", `\=`, "!", `\!`,
)
var markdownBlockMarker = regexp.MustCompile(`^([ \t]*\d+)([.)])([ \t]|$)`)

func markdownText(text string) string {
	text = markdownLiteral.Replace(text)
	return markdownBlockMarker.ReplaceAllString(text, `${1}\${2}${3}`)
}

func markdownDestination(source string) string {
	return strings.NewReplacer(
		" ", "%20", "(", "%28", ")", "%29", "<", "%3C", ">", "%3E",
		"\n", "%0A", "\r", "%0D", "\t", "%09", `\`, "%5C",
	).Replace(source)
}

func markdownEmphasis(text, marker string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return text
	}
	start := strings.Index(text, trimmed)
	return text[:start] + marker + trimmed + marker + text[start+len(trimmed):]
}

func renderText(root *html.Node, markdown bool) string {
	var render func(*html.Node) string
	children := func(node *html.Node) string {
		var output strings.Builder
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			if child.Type == html.TextNode && strings.TrimSpace(child.Data) == "" &&
				(isBlock(child.PrevSibling) || isBlock(child.NextSibling)) {
				continue
			}
			output.WriteString(render(child))
		}

		return output.String()
	}
	render = func(node *html.Node) string {
		if node.Type == html.TextNode {
			text := space.ReplaceAllString(node.Data, " ")
			if markdown {
				return markdownText(text)
			}
			return text
		}
		switch node.Data {
		case "pre":
			code := strings.TrimRight(nodeText(node, true), "\n")
			fence := "`"
			for strings.Contains(code, fence) || len(fence) < 3 {
				fence += "`"
			}
			return fence + "\n" + code + "\n" + fence + "\n\n"
		case "code":
			text := nodeText(node, true)
			if markdown && text == "" {
				return "<code></code>"
			}
			fence := "`"
			for strings.Contains(text, fence) {
				fence += "`"
			}
			pad := strings.Contains(text, "`") || strings.HasPrefix(text, " ") || strings.HasSuffix(text, " ")
			if markdown && strings.Trim(text, " ") == "" {
				pad = false
			}
			if pad {
				text = " " + text + " "
			}
			return fence + text + fence
		case "table":
			if markdown {
				rows := tableRows(node)
				width := 0
				for _, row := range rows {
					width = max(width, len(row))
				}
				var lines []string
				markdownRow := func(cells []string) string {
					return "| " + strings.Join(cells, " | ") + " |"
				}
				separators := make([]string, width)
				for index := range separators {
					separators[index] = "---"
				}
				for index, row := range rows {
					cells := make([]string, width)
					for column, cell := range row {
						value := strings.TrimSpace(children(cell))
						cells[column] = strings.NewReplacer("|", `\|`, "  \n", "<br>", "\n", "<br>").Replace(value)
					}
					if index == 0 && len(row) > 0 && row[0].Data != "th" {
						lines = append(lines, markdownRow(make([]string, width)), markdownRow(separators))
					}
					lines = append(lines, markdownRow(cells))
					if index == 0 && len(row) > 0 && row[0].Data == "th" {
						lines = append(lines, markdownRow(separators))
					}
				}
				return strings.Join(lines, "\n") + "\n\n"
			}
			var lines []string
			for _, row := range tableRows(node) {
				var cells []string
				for _, cell := range row {
					cells = append(cells, strings.TrimSpace(nodeText(cell, true)))
				}
				lines = append(lines, strings.Join(cells, "\t"))
			}
			return strings.Join(lines, "\n") + "\n\n"
		case "ul", "ol":
			number, _ := strconv.Atoi(attr(node, "start"))
			if number <= 0 {
				number = 1
			}
			var lines []string
			for child := node.FirstChild; child != nil; child = child.NextSibling {
				if child.Data != "li" {
					continue
				}
				marker := "- "
				if node.Data == "ol" {
					marker = fmt.Sprintf("%d. ", number)
					number++
				}
				content := strings.TrimSpace(children(child))
				content = strings.ReplaceAll(content, "\n", "\n"+strings.Repeat(" ", len(marker)))
				lines = append(lines, marker+content)
			}
			prefix := ""
			if node.Parent != nil && node.Parent.Data == "li" {
				prefix = "\n"
			}
			return prefix + strings.Join(lines, "\n") + "\n\n"
		case "img":
			source := attr(node, "data-copy-source")
			if source == "" {
				source = attr(node, "src")
			}
			alt := strings.ReplaceAll(attr(node, "alt"), "]", `\]`)
			if markdown {
				alt = markdownText(attr(node, "alt"))
			}
			source = markdownDestination(source)
			return "![" + alt + "](" + source + ")"
		case "br":
			if markdown {
				return "  \n"
			}
			return "\n"
		case "hr":
			return "---\n\n"
		}
		text := children(node)
		switch node.Data {
		case "strong":
			if markdown {
				return markdownEmphasis(text, "**")
			}
			return "**" + text + "**"
		case "em":
			if markdown {
				return markdownEmphasis(text, "*")
			}
			return "*" + text + "*"
		case "u":
			if markdown {
				return "<u>" + text + "</u>"
			}
			return "__" + text + "__"
		case "sup", "sub":
			if markdown {
				return "<" + node.Data + ">" + text + "</" + node.Data + ">"
			}
		case "del":
			if markdown {
				return markdownEmphasis(text, "~~")
			}
			return "~~" + text + "~~"
		case "a":
			if href := attr(node, "href"); href != "" {
				if markdown {
					return "[" + text + "](" + markdownDestination(href) + ")"
				}
				return "[" + text + "](" + strings.ReplaceAll(href, ")", "%29") + ")"
			}
		case "h1", "h2", "h3", "h4", "h5", "h6":
			return strings.Repeat("#", int(node.Data[1]-'0')) + " " + strings.TrimSpace(text) + "\n\n"
		case "blockquote":
			return "> " + strings.ReplaceAll(strings.TrimSpace(text), "\n", "\n> ") + "\n\n"
		case "p", "div":
			if strings.TrimSpace(text) != "" {
				return strings.Trim(text, "\n") + "\n\n"
			}
		}
		return text
	}
	result := children(root)
	if !markdown && root.FirstChild != nil && root.FirstChild == root.LastChild && root.FirstChild.Data == "table" {
		return strings.TrimSuffix(result, "\n")
	}
	return strings.Trim(result, "\n")
}
