package copy_format

import (
	"encoding/binary"
	"encoding/json"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"unicode/utf16"

	"golang.org/x/net/html"
)

func TestSlackListItemsHaveSingleLineSpacing(t *testing.T) {
	for _, tag := range []string{"ul", "ol"} {
		t.Run(tag, func(t *testing.T) {
			source := "<" + tag + ">\n<li>\n<p>First</p>\n</li>\n\n<li><p>Second</p>\n<ul>\n<li>Nested</li>\n</ul>\n</li>\n<li>Third</li>\n</" + tag + ">"
			formats, err := Convert(source)
			if err != nil {
				t.Fatal(err)
			}
			want := "- <span>First</span><br/>- <span>Second</span><br/>  - Nested<br/>- Third<br/>"
			if tag == "ol" {
				want = "1. <span>First</span><br/>2. <span>Second</span><br/>  - Nested<br/>3. Third<br/>"
			}
			if !strings.Contains(formats.SlackHTML, want) {
				t.Fatalf("list spacing mismatch: %s", formats.SlackHTML)
			}
			if strings.Contains(formats.SlackHTML, "<br/><br/>") {
				t.Fatal("blank line between list items")
			}
			if !strings.Contains(formats.HTML, "<"+tag+">") || !strings.Contains(formats.HTML, "\n\n") {
				t.Fatal("Slack spacing change modified standard HTML")
			}
		})
	}
}

func TestSlackPreservesParagraphsAndEmphasis(t *testing.T) {
	formats, err := Convert(`<div><p>First <em>italic</em></p><p>Second</p><ul><li><p>Item <em>text</em></p></li></ul></div>`)
	if err != nil {
		t.Fatal(err)
	}
	for _, tag := range []string{"<p>", "</p>"} {
		if !strings.Contains(formats.SlackHTML, tag) {
			t.Errorf("Slack HTML missing %s", tag)
		}
	}
	if !strings.Contains(formats.SlackHTML, "<p>First <span>_italic_</span></p><p>Second</p>- <span>Item <span>_text_</span></span><br/>") {
		t.Fatalf("Slack paragraph content or separation changed: %s", formats.SlackHTML)
	}
	if !strings.Contains(formats.HTML, "<p>First <em>italic</em></p>") ||
		!strings.Contains(formats.Markdown, "*italic*") || !strings.Contains(formats.RTF, `{\i italic}`) {
		t.Fatal("Slack experiment changed other representations")
	}
}

func TestSlackHybridTextListsAndTableFormatting(t *testing.T) {
	formats, err := Convert(`<h2>Heading</h2><p><strong>Bold</strong> <i>Italic</i> <u>Underline</u> <code>*literal*</code></p><ol start="4"><li>Four<ul><li>Nested</li></ul></li><li><p>Five</p></li></ol><table><tr><th>Title</th></tr><tr><td><strong>Bold</strong> <em>Italic</em><ul><li>Table list</li></ul></td></tr></table>`)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`<p><br/><br/>## _Heading_<br/><br/></p>`,
		`<span>*Bold*</span> <span>_Italic_</span> <span>_Underline_</span> <code>*literal*</code>`,
		`4. Four<br/>  - Nested<br/>5. <span>Five</span><br/>`,
		`<td><b data-stringify-type="bold">Bold</b> <em>Italic</em><ul><li>Table list</li></ul></td>`,
	} {
		if !strings.Contains(formats.SlackHTML, want) {
			t.Errorf("Slack hybrid output missing %q: %s", want, formats.SlackHTML)
		}
	}
	root, err := html.Parse(strings.NewReader(formats.SlackHTML))
	if err != nil {
		t.Fatal(err)
	}
	var visit func(*html.Node, bool)
	visit = func(node *html.Node, inTable bool) {
		inTable = inTable || node.Data == "table"
		if !inTable && node.Type == html.ElementNode {
			switch node.Data {
			case "h1", "h2", "h3", "h4", "h5", "h6", "ul", "ol", "li", "strong", "b", "em", "i", "u":
				t.Errorf("unexpected formatting tag outside table: %s", node.Data)
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child, inTable)
		}
	}
	visit(root, false)
	if !strings.Contains(formats.HTML, `<ol start="4"><li>Four<ul>`) ||
		!strings.Contains(formats.HTML, "<h2>Heading</h2>") ||
		!strings.Contains(formats.Markdown, "## Heading") ||
		!strings.Contains(formats.Markdown, "**Bold**") {
		t.Fatal("Slack hybrid formatting affected global representations")
	}
}

func TestSlackHeadingsUseMarkdownHashes(t *testing.T) {
	for level := 1; level <= 6; level++ {
		tag := "h" + strconv.Itoa(level)
		t.Run(tag, func(t *testing.T) {
			formats, err := Convert("<" + tag + ">Title <strong>bold</strong></" + tag + ">")
			if err != nil {
				t.Fatal(err)
			}
			want := "<p><br/><br/>" + strings.Repeat("#", level) + ` _Title bold_<br/><br/></p>`
			if !strings.Contains(formats.SlackHTML, want) || strings.Contains(formats.SlackHTML, "<"+tag+">") {
				t.Fatalf("Slack heading = %s, want %s", formats.SlackHTML, want)
			}
			if !strings.Contains(formats.HTML, "<"+tag+">") {
				t.Fatal("Slack heading conversion changed standard HTML")
			}
			if formats.Markdown != strings.Repeat("#", level)+" Title **bold**" {
				t.Fatalf("Markdown heading changed: %q", formats.Markdown)
			}
		})
	}
}

func TestSlackListsUnwrapUnclassedDivs(t *testing.T) {
	formats, err := Convert(`<div><div><p>Before</p><div><ul><li>First</li><li><strong>Second</strong><div><ul><li>Nested</li></ul></div></li></ul></div><p>After</p></div></div>`)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(formats.SlackHTML, "<div>") {
		t.Fatalf("Slack HTML retained unclassed divs: %s", formats.SlackHTML)
	}
	for _, want := range []string{
		`<div class="p-rich_text_section"><p>Before</p>- First<br/>`,
		`- <span>*Second*</span>`,
		`  - Nested<br/><p>After</p></div>`,
	} {
		if !strings.Contains(formats.SlackHTML, want) {
			t.Errorf("Slack HTML missing %q: %s", want, formats.SlackHTML)
		}
	}
	if !strings.Contains(formats.HTML, "<div><div>") {
		t.Fatal("standard HTML rendering divs changed")
	}
}

func TestSlackBoldUsesNativeMarkupWithoutMarkdownDelimiters(t *testing.T) {
	formats, err := Convert(`<p><strong>Bold <em>nested</em></strong> and literal *stars*</p><table><tr><td><strong>Bold cell</strong></td></tr></table>`)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		`<span>*Bold <span>_nested_</span>*</span>`,
		`<b data-stringify-type="bold">Bold cell</b>`,
		`literal *stars*`,
	} {
		if !strings.Contains(formats.SlackHTML, want) {
			t.Errorf("Slack HTML missing %q: %s", want, formats.SlackHTML)
		}
	}
	for _, unwanted := range []string{"<strong>", "**Bold", "*Bold cell*"} {
		if strings.Contains(formats.SlackHTML, unwanted) {
			t.Errorf("Slack HTML contains %q", unwanted)
		}
	}
	if !strings.Contains(formats.HTML, "<strong>Bold <em>nested</em></strong>") ||
		strings.Contains(formats.HTML, "data-stringify-type") {
		t.Fatal("Slack bold adaptation changed standard HTML")
	}
	if !strings.Contains(formats.Markdown, "**Bold *nested***") {
		t.Fatal("Markdown bold formatting was removed")
	}
}

func TestSlackDocumentBookmarksAreTextNotLinks(t *testing.T) {
	source := `<p><a href="#section"><strong>Section</strong></a> <a href="#">Top</a> <a href="https://example.com/doc#section">External</a></p><table><tr><td><a href="#section"><em>Jump</em></a></td><td><a href="https://example.com/#target">Website</a></td></tr></table>`
	formats, err := Convert(source)
	if err != nil {
		t.Fatal(err)
	}
	for _, unwanted := range []string{`href="#`, `&#34;url&#34;:&#34;#`, `"url":"#`} {
		if strings.Contains(formats.SlackHTML, unwanted) {
			t.Errorf("Slack retained document bookmark link %q: %s", unwanted, formats.SlackHTML)
		}
	}
	for _, want := range []string{
		`<span><span>*Section*</span></span>`,
		`<span>Top</span>`,
		`<span><em>Jump</em></span>`,
		`href="https://example.com/doc#section"`,
		`href="https://example.com/#target"`,
	} {
		if !strings.Contains(formats.SlackHTML, want) {
			t.Errorf("Slack HTML missing %q", want)
		}
	}
	nodes, err := html.ParseFragment(strings.NewReader(source), nil)
	if err != nil {
		t.Fatal(err)
	}
	var visit func(*html.Node)
	visit = func(node *html.Node) {
		if node.Data == "td" && strings.Contains(nodeText(node, true), "Jump") {
			want := []slackInline{{Type: "text", Text: "Jump", Style: &slackStyle{Italic: true}}}
			if got := slackCellElements(node); !reflect.DeepEqual(got, want) {
				t.Errorf("bookmark cell elements = %+v, want %+v", got, want)
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	for _, node := range nodes {
		visit(node)
	}
	if !strings.Contains(formats.HTML, `href="#section"`) || !strings.Contains(formats.Markdown, "(#section)") ||
		!strings.Contains(formats.RTF, `HYPERLINK "#section"`) {
		t.Fatal("bookmark adaptation changed non-Slack formats")
	}
}

func TestSlackHTMLIncludesTableSourceWithoutMessageMetadata(t *testing.T) {
	formats, err := Convert(`<div data-message-channel="not-for-export"><p>Before</p>
<table>
<tr><th>Label</th><th>Value</th></tr>
<tr><td>Second 😀</td><td data-formula="=A1+1">42</td></tr>
<tr><td><strong>Bold</strong> <em>italic</em> <a href="https://example.com/?a=1&amp;b=2">link</a></td><td></td></tr>
</table>
<p>After</p></div>`)
	if err != nil {
		t.Fatal(err)
	}
	root, err := html.Parse(strings.NewReader(formats.SlackHTML))
	if err != nil {
		t.Fatal(err)
	}
	var blocks []*html.Node
	var visit func(*html.Node)
	visit = func(node *html.Node) {
		if attr(node, "data-block-type") != "" {
			blocks = append(blocks, node)
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(root)
	if len(blocks) != 3 {
		t.Fatalf("Slack block count = %d, want 3: %s", len(blocks), formats.SlackHTML)
	}
	for index, want := range []string{"rich_text", "table", "rich_text"} {
		if attr(blocks[index], "data-block-type") != want {
			t.Errorf("block %d type = %s, want %s", index, attr(blocks[index], "data-block-type"), want)
		}
	}
	var table slackTable
	if err := json.Unmarshal([]byte(attr(blocks[1], "data-block-source")), &table); err != nil {
		t.Fatal(err)
	}
	if table.Type != "table" || len(table.Rows) != 3 || len(table.Rows[0]) != 2 {
		t.Fatalf("unexpected table structure: %+v", table)
	}
	header := table.Rows[0][0]
	if header.Type != "rich_text" || len(header.Elements) != 1 ||
		!reflect.DeepEqual(header.Elements[0].Elements, []slackInline{{Type: "text", Text: "Label", Style: &slackStyle{Bold: true}}}) {
		t.Errorf("header = %+v", header)
	}
	for column, want := range []string{"Second 😀", "42"} {
		cell := table.Rows[1][column]
		if cell.Type != "raw_text" || cell.Text == nil || *cell.Text != want {
			t.Errorf("body cell %d = %+v, want %q", column, cell, want)
		}
	}
	wantInline := []slackInline{
		{Type: "text", Text: "Bold", Style: &slackStyle{Bold: true}},
		{Type: "text", Text: " "},
		{Type: "text", Text: "italic", Style: &slackStyle{Italic: true}},
		{Type: "text", Text: " "},
		{Type: "link", Text: "link", URL: "https://example.com/?a=1&b=2"},
	}
	if got := table.Rows[2][0].Elements[0].Elements; !reflect.DeepEqual(got, wantInline) {
		t.Errorf("formatted cell = %+v, want %+v", got, wantInline)
	}
	if cell := table.Rows[2][1]; cell.Type != "raw_text" || cell.Text == nil || *cell.Text != "-" {
		t.Errorf("empty cell = %+v", cell)
	}
	for _, unwanted := range []string{"data-message-channel", "data-message-ts", "data-block-id", "data-formula", "not-for-export"} {
		if strings.Contains(formats.SlackHTML, unwanted) {
			t.Errorf("Slack HTML retained %q", unwanted)
		}
	}
	if strings.Contains(formats.HTML, "data-block-source") {
		t.Fatal("standard HTML was changed to Slack markup")
	}
}

func TestSlackEmptyCellsBecomeDashesOnlyInSlack(t *testing.T) {
	source := "<table><tr><th></th><th><strong> \u00a0<br></strong></th></tr><tr><td></td><td><p> \t<br></p></td></tr><tr><td>0</td><td>kept</td></tr></table>"
	formats, err := Convert(source)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(formats.SlackHTML, "<th>-</th>") != 2 || strings.Count(formats.SlackHTML, "<td>-</td>") != 2 {
		t.Fatalf("Slack HTML empty cells were not replaced: %s", formats.SlackHTML)
	}
	root, err := html.Parse(strings.NewReader(formats.SlackHTML))
	if err != nil {
		t.Fatal(err)
	}
	var visit func(*html.Node)
	found := false
	visit = func(node *html.Node) {
		if attr(node, "data-block-type") == "table" {
			found = true
			var table slackTable
			if err := json.Unmarshal([]byte(attr(node, "data-block-source")), &table); err != nil {
				t.Fatal(err)
			}
			for _, cell := range table.Rows[0] {
				if cell.Type != "rich_text" || len(cell.Elements) != 1 ||
					!reflect.DeepEqual(cell.Elements[0].Elements, []slackInline{{Type: "text", Text: "-", Style: &slackStyle{Bold: true}}}) {
					t.Errorf("empty heading metadata = %+v", cell)
				}
			}
			for _, cell := range table.Rows[1] {
				if cell.Type != "raw_text" || cell.Text == nil || *cell.Text != "-" {
					t.Errorf("empty body metadata = %+v", cell)
				}
			}
			for index, want := range []string{"0", "kept"} {
				if cell := table.Rows[2][index]; cell.Text == nil || *cell.Text != want {
					t.Errorf("nonempty cell changed: %+v", cell)
				}
			}
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
	}
	visit(root)
	if !found {
		t.Fatal("Slack table metadata missing")
	}
	if !strings.Contains(formats.HTML, "<th></th>") || !strings.Contains(formats.HTML, "<td></td>") ||
		strings.Contains(formats.PlainText, "-") || strings.Contains(formats.RTF, `-\cell`) ||
		strings.Contains(formats.Markdown, "| - |") {
		t.Fatal("Slack placeholders leaked into other formats")
	}
}

func TestSlackCellLineBreaksAndStyles(t *testing.T) {
	formats, err := Convert(`<table><tr><td><p>One</p><p>Two<br>Three</p><pre><code>a  b
c</code></pre><del>gone</del></td></tr></table>`)
	if err != nil {
		t.Fatal(err)
	}
	root, err := html.Parse(strings.NewReader(formats.HTML))
	if err != nil {
		t.Fatal(err)
	}
	var cell *html.Node
	var find func(*html.Node)
	find = func(node *html.Node) {
		if node.Data == "td" {
			cell = node
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			find(child)
		}
	}
	find(root)
	got := slackCellElements(cell)
	want := []slackInline{
		{Type: "text", Text: "One\nTwo\nThree\n"},
		{Type: "text", Text: "a  b\nc", Style: &slackStyle{Code: true}},
		{Type: "text", Text: "\n"},
		{Type: "text", Text: "gone", Style: &slackStyle{Strike: true}},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Slack cell = %+v, want %+v", got, want)
	}
}

func TestMarkdownRepresentationKeepsTableStructure(t *testing.T) {
	for _, test := range []struct {
		name, source, markdown, plain string
	}{
		{"headers", `<table><tr><th>Name</th><th>Value</th></tr><tr><td>A|B</td><td><strong>2</strong><br>3</td></tr></table>`,
			"| Name | Value |\n| --- | --- |\n| A\\|B | **2**<br>3 |", "Name\tValue\nA|B\t2\n3\n"},
		{"partial", `<table><tr><td>first</td><td></td></tr><tr><td>second</td></tr></table>`,
			"|  |  |\n| --- | --- |\n| first |  |\n| second |  |", "first\t\nsecond\n"},
		{"empty", `<table></table>`, "", "\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			formats, err := Convert(test.source)
			if err != nil {
				t.Fatal(err)
			}
			if formats.Markdown != test.markdown || formats.PlainText != test.plain {
				t.Errorf("Markdown = %q, plain = %q; want %q and %q", formats.Markdown, formats.PlainText, test.markdown, test.plain)
			}
		})
	}
}

func TestChromiumDataUTF16AndAlignment(t *testing.T) {
	formats := Formats{HTML: "<p>é 😀</p>", SlackHTML: "<div>Slack</div>", Markdown: "# odd"}
	data := ChromiumData(formats)
	if len(data)%4 != 0 || int(binary.LittleEndian.Uint32(data)) != len(data)-4 {
		t.Fatal("invalid Pickle size or alignment")
	}
	if count := binary.LittleEndian.Uint32(data[4:]); count != 3 {
		t.Fatalf("entry count = %d, want 3", count)
	}
	offset := 8
	readString := func() string {
		t.Helper()
		if offset+4 > len(data) {
			t.Fatal("missing string length")
		}
		length := int(binary.LittleEndian.Uint32(data[offset:]))
		offset += 4
		if length > (len(data)-offset)/2 {
			t.Fatal("invalid UTF-16 length")
		}
		units := make([]uint16, length)
		for index := range units {
			units[index] = binary.LittleEndian.Uint16(data[offset+index*2:])
		}
		end := offset + length*2
		offset = (end + 3) &^ 3
		for _, padding := range data[end:offset] {
			if padding != 0 {
				t.Fatal("nonzero Pickle padding")
			}
		}
		return string(utf16.Decode(units))
	}
	got := map[string]string{}
	for range 3 {
		key := readString()
		got[key] = readString()
	}
	want := map[string]string{"slack/html": formats.SlackHTML, "text/html": formats.HTML, "text/markdown": formats.Markdown}
	if !reflect.DeepEqual(got, want) || offset != len(data) {
		t.Fatalf("decoded custom data = %v, want %v; consumed %d/%d bytes", got, want, offset, len(data))
	}
	empty := ChromiumData(Formats{})
	if len(empty) != 100 || binary.LittleEndian.Uint32(empty) != 96 {
		t.Fatalf("empty-value Pickle = %d bytes, want fixed 100-byte encoding", len(empty))
	}
}
