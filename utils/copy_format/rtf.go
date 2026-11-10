package copy_format

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf16"

	"golang.org/x/net/html"
)

func escapeRTF(text string) string {
	var output strings.Builder
	for _, r := range text {
		switch r {
		case '\\', '{', '}':
			output.WriteByte('\\')
			output.WriteRune(r)
		case '\n':
			output.WriteString(`\line `)
		case '\t':
			output.WriteString(`\tab `)
		default:
			if r >= 128 {
				for _, unit := range utf16.Encode([]rune{r}) {
					fmt.Fprintf(&output, `\u%d?`, int16(unit))
				}
			} else if r >= 32 {
				output.WriteRune(r)
			}
		}
	}
	return output.String()
}

func renderRTF(root *html.Node) string {
	var render func(*html.Node, bool) string
	children := func(node *html.Node, preserve bool) string {
		var output strings.Builder
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			output.WriteString(render(child, preserve))
		}
		return output.String()
	}
	render = func(node *html.Node, preserve bool) string {
		if node.Type == html.TextNode {
			text := node.Data
			if !preserve {
				text = space.ReplaceAllString(text, " ")
			}
			return escapeRTF(text)
		}
		switch node.Data {
		case "img":
			data, config, _ := pngData(attr(node, "src")) // Validated by clean.
			width, height := config.Width*15, config.Height*15
			if width > 9000 {
				height = height * 9000 / width
				width = 9000
			}
			return fmt.Sprintf(`{\pict\pngblip\picw%d\pich%d\picwgoal%d\pichgoal%d %s}`, config.Width, config.Height, width, height, hex.EncodeToString(data))
		case "table":
			rows := tableRows(node)
			columns := 0
			for _, row := range rows {
				columns = max(columns, len(row))
			}
			width := max(1, 9000/max(1, columns))
			var output strings.Builder
			for index, row := range rows {
				output.WriteString(`\trowd\trgaph108\trleft0`)
				if index == 0 {
					output.WriteString(`\trhdr`)
				}
				for column := 0; column < columns; column++ {
					fmt.Fprintf(&output, `\cellx%d`, (column+1)*width)
				}
				for column := 0; column < columns; column++ {
					output.WriteString(`\pard\intbl `)
					if index == 0 {
						output.WriteString(`\b `)
					}
					if column < len(row) {
						output.WriteString(children(row[column], true))
					}
					if index == 0 {
						output.WriteString(`\b0`)
					}
					output.WriteString(`\cell `)
				}
				output.WriteString("\\row\n")
			}
			return output.String() + "\\pard\n"
		case "ul", "ol":
			number, _ := strconv.Atoi(attr(node, "start"))
			if number <= 0 {
				number = 1
			}
			var output strings.Builder
			for child := node.FirstChild; child != nil; child = child.NextSibling {
				if child.Data != "li" {
					continue
				}
				marker := `\bullet`
				if node.Data == "ol" {
					marker = fmt.Sprintf("%d.", number)
					number++
				}
				fmt.Fprintf(&output, `{\pard\li360\fi-180 %s\tab %s\par}`, marker, children(child, false))
			}
			return output.String()
		case "pre":
			return `{\pard\f1 ` + escapeRTF(nodeText(node, true)) + `\par}`
		case "br":
			return `\line `
		case "hr":
			return `{\pard\brdrb\brdrs\brdrw10\par}`
		}
		text := children(node, preserve)
		switch node.Data {
		case "strong":
			return `{\b ` + text + "}"
		case "em":
			return `{\i ` + text + "}"
		case "u":
			return `{\ul ` + text + "}"
		case "del":
			return `{\strike ` + text + "}"
		case "code":
			return `{\f1 ` + text + "}"
		case "sup":
			return `{\super ` + text + "}"
		case "sub":
			return `{\sub ` + text + "}"
		case "h1", "h2", "h3", "h4", "h5", "h6":
			return fmt.Sprintf(`{\pard\b\fs%d %s\par}`, 40-3*int(node.Data[1]-'1'), text)
		case "p", "div":
			return text + "\\par\n"
		case "blockquote":
			return `{\pard\li360 ` + text + `\par}`
		case "a":
			if href := attr(node, "href"); href != "" {
				href = strings.ReplaceAll(href, `"`, "%22")
				return `{\field{\*\fldinst HYPERLINK "` + escapeRTF(href) + `"}{\fldrslt ` + text + `}}`
			}
		}
		return text
	}
	return "{\\rtf1\\ansi\\ansicpg1252\\deff0\\uc1{\\fonttbl{\\f0\\fswiss Helvetica;}{\\f1\\fmodern Courier New;}}\\f0\\fs20\n" + children(root, false) + "}"
}
