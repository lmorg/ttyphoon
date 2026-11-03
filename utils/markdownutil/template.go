package markdownutil

import (
	"html"
	"strings"
	"text/template"
)

const markdownEscapedPunctuation = "\\`*_{}[]()#+-.!|~"

func TemplateFuncMap() template.FuncMap {
	return template.FuncMap{
		"escapeMarkdown":    EscapeMarkdownText,
		"markdownCodeBlock": MarkdownCodeBlock,
		"markdownTableCode": MarkdownTableCode,
		"add":               Add,
	}
}

func EscapeMarkdownText(value string) string {
	value = strings.NewReplacer("\r\n", " ", "\r", " ", "\n", " ").Replace(value)
	var escaped strings.Builder
	escaped.Grow(len(value))
	for _, character := range value {
		switch character {
		case '&', '<', '>', '"', '\'':
			escaped.WriteString(html.EscapeString(string(character)))
		default:
			if strings.ContainsRune(markdownEscapedPunctuation, character) {
				escaped.WriteByte('\\')
			}
			escaped.WriteRune(character)
		}
	}
	return escaped.String()
}

func MarkdownCodeBlock(value string, language ...string) string {
	longestBacktickRun := 0
	currentBacktickRun := 0
	for _, character := range value {
		if character == '`' {
			currentBacktickRun++
			if currentBacktickRun > longestBacktickRun {
				longestBacktickRun = currentBacktickRun
			}
			continue
		}
		currentBacktickRun = 0
	}

	fence := strings.Repeat("`", max(3, longestBacktickRun+1))
	info := ""
	if len(language) > 0 {
		info = strings.TrimSpace(language[0])
	}
	openingFence := fence
	if info != "" {
		openingFence += info
	}

	var block strings.Builder
	block.Grow(len(value) + len(fence)*2 + len(info) + 3)
	block.WriteString(openingFence)
	block.WriteByte('\n')
	block.WriteString(value)
	if !strings.HasSuffix(value, "\n") {
		block.WriteByte('\n')
	}
	block.WriteString(fence)
	return block.String()
}

func Add(value, amount int) int {
	return value + amount
}

// Table rows must stay on one line, so newlines become &#10; and Markdown syntax is entity-encoded.
var markdownTableCodeEscaper = strings.NewReplacer(
	"&", "&amp;",
	"<", "&lt;",
	">", "&gt;",
	"|", "&#124;",
	"\\", "&#92;",
	"`", "&#96;",
	"*", "&#42;",
	"_", "&#95;",
	"~", "&#126;",
	"[", "&#91;",
	"]", "&#93;",
	"$", "&#36;",
	"\r", "",
	"\n", "&#10;",
)

func MarkdownTableCode(value string) string {
	return "<pre><code>" + markdownTableCodeEscaper.Replace(value) + "</code></pre>"
}
