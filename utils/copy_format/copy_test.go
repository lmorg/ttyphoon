package copy_format

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
)

func TestSemanticFormats(t *testing.T) {
	source := `<h2>Title</h2><p>Hello <strong>bold</strong>, <em>italic</em>, <u>underlined</u>, <code>fixed</code>.</p><pre><code class="language-go">x := 1
print(x)</code></pre><ol start="3"><li>Three</li><li>Four</li></ol><ul><li>Bullet</li></ul>`
	got, err := Convert(source)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"<h2>Title</h2>", "<strong>bold</strong>", "<em>italic</em>", "<u>underlined</u>", "<pre><code>x := 1"} {
		if !strings.Contains(got.HTML, want) {
			t.Errorf("HTML missing %q: %s", want, got.HTML)
		}
	}
	for _, want := range []string{"## Title", "**bold**", "*italic*", "__underlined__", "`fixed`", "```\nx := 1\nprint(x)\n```", "3. Three\n4. Four", "- Bullet"} {
		if !strings.Contains(got.PlainText, want) {
			t.Errorf("plain text missing %q: %s", want, got.PlainText)
		}
	}
	for _, want := range []string{`{\rtf1`, `{\b bold}`, `{\i italic}`, `{\ul underlined}`, `{\f1 fixed}`, `\fs37 Title`, `3.\tab Three`, `\bullet\tab Bullet`} {
		if !strings.Contains(got.RTF, want) {
			t.Errorf("RTF missing %q: %s", want, got.RTF)
		}
	}
	if strings.Contains(got.PlainText, "```go") || strings.Contains(got.HTML, "language-go") {
		t.Fatal("export retained language metadata")
	}
}

func TestTableFormatsUseDisplayedCellValues(t *testing.T) {
	got, err := Convert(`<table><thead><tr><th>Name</th><th>Value</th></tr></thead><tbody><tr><td>Beta</td><td>2<br/>3</td></tr></tbody></table>`)
	if err != nil {
		t.Fatal(err)
	}
	if got.PlainText != "Name\tValue\nBeta\t2\n3\n" {
		t.Errorf("table plain text = %q", got.PlainText)
	}
	for _, want := range []string{`\trowd\trgaph108\trleft0\trhdr\cellx4500\cellx9000`, `\b Name\b0\cell`, `Beta\cell`, `2\line 3\cell`} {
		if !strings.Contains(got.RTF, want) {
			t.Errorf("table RTF missing %q: %s", want, got.RTF)
		}
	}
	if !strings.Contains(got.HTML, "<td>2<br/>3</td>") {
		t.Errorf("table HTML = %s", got.HTML)
	}
}

func TestImagesAreInlineInHTMLAndRTF(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 2, 1))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, img); err != nil {
		t.Fatal(err)
	}
	source := "data:image/png;base64," + base64.StdEncoding.EncodeToString(buffer.Bytes())
	got, err := Convert(`<p><img src="` + source + `" alt="A &amp; B" data-copy-source="images/test.png"></p>`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got.HTML, source) || !strings.Contains(got.HTML, `alt="A &amp; B"`) {
		t.Errorf("inline HTML = %s", got.HTML)
	}
	if strings.Contains(got.HTML, "data-copy-source") {
		t.Fatal("HTML includes internal image metadata")
	}
	if !strings.Contains(got.RTF, `\pict\pngblip\picw2\pich1\picwgoal30\pichgoal15 89504e47`) {
		t.Errorf("inline RTF = %s", got.RTF)
	}
	if got.PlainText != "![A & B](images/test.png)" {
		t.Errorf("image plain text = %q", got.PlainText)
	}
	if got.Markdown != "![A &amp; B](images/test.png)" {
		t.Errorf("image Markdown = %q", got.Markdown)
	}
}

func TestEscapingAndSafeHTML(t *testing.T) {
	got, err := Convert(`<p onclick="evil()">café 😀 \ { } &lt;x&gt;<script>evil()</script><a href="javascript:evil()">bad</a><a href="https://example.com/">good</a></p>`)
	if err != nil {
		t.Fatal(err)
	}
	for _, unwanted := range []string{"onclick", "<script", "evil()", "javascript:"} {
		if strings.Contains(got.HTML, unwanted) {
			t.Errorf("HTML contains %q: %s", unwanted, got.HTML)
		}
	}
	for _, want := range []string{`\u233?`, `\u-10179?\u-8704?`, `\\ \{ \}`, `HYPERLINK "https://example.com/"`} {
		if !strings.Contains(got.RTF, want) {
			t.Errorf("escaped RTF missing %q: %s", want, got.RTF)
		}
	}
}

func TestCodeFencesPreserveEmbeddedBackticks(t *testing.T) {
	got, err := PlainText("<pre><code>```code```</code></pre><p><code>a`b</code></p>")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "````\n```code```\n````") || !strings.Contains(got, "`` a`b ``") {
		t.Errorf("code fences = %q", got)
	}
}

func TestNestedListsAndWhitespaceBetweenBlocks(t *testing.T) {
	got, err := PlainText("<h1>Title</h1>\n<p>Body <em>one</em> <strong>two</strong></p>\n<ul><li>Parent<ul><li>Child</li></ul></li><li>Next</li></ul>")
	if err != nil {
		t.Fatal(err)
	}
	want := "# Title\n\nBody *one* **two**\n\n- Parent\n  - Child\n- Next"
	if got != want {
		t.Errorf("plain text = %q, want %q", got, want)
	}
}

func TestTableRTFUnicodeAndControlCharacters(t *testing.T) {
	got, err := Convert("<table><tr><th>Name</th></tr><tr><td>{value}\\path\tCafé😀</td></tr></table>")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{`\{value\}\\path\tab Caf\u233?\u-10179?\u-8704?`, `\trhdr`, `\b Name\b0`} {
		if !strings.Contains(got.RTF, want) {
			t.Errorf("table RTF missing %q: %s", want, got.RTF)
		}
	}
	if got.PlainText != "Name\n{value}\\path\tCafé😀\n" {
		t.Errorf("table text = %q", got.PlainText)
	}
}

func TestEmptyTableCellsAndRowsArePreserved(t *testing.T) {
	got, err := Convert("<table><tr><th>Name</th></tr><tr><td></td></tr><tr><td></td></tr></table>")
	if err != nil {
		t.Fatal(err)
	}
	if got.PlainText != "Name\n\n\n" || strings.Count(got.RTF, `\row`) != 3 || strings.Count(got.HTML, "<td></td>") != 2 {
		t.Fatalf("empty rows changed: %+v", got)
	}
}

func TestRejectInvalidImagesAndExcessiveDepth(t *testing.T) {
	for _, source := range []string{
		`<img src="https://example.com/image.png">`,
		`<img src="data:image/png;base64,invalid">`,
		strings.Repeat("<div>", 258) + "deep" + strings.Repeat("</div>", 258),
	} {
		if _, err := Convert(source); err == nil {
			t.Errorf("accepted invalid source %.80q", source)
		}
	}
}
