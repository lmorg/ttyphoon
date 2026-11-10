package copy_format

import (
	"encoding/binary"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"unicode/utf16"

	"golang.org/x/net/html"
)

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
	if cell := table.Rows[2][1]; cell.Type != "raw_text" || cell.Text == nil || *cell.Text != "" {
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
