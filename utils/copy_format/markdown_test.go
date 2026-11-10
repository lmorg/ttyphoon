package copy_format

import "testing"

func TestMarkdownSelectionElements(t *testing.T) {
	for _, test := range []struct {
		name, source, want string
	}{
		{"heading", `<h2>Title *literal*</h2>`, `## Title \*literal\*`},
		{"inline", `<p><strong>bold</strong>, <em>italic</em>, <u>underlined</u>, <del>removed</del>, <code>*fixed*</code>.</p>`,
			"**bold**, *italic*, <u>underlined</u>, ~~removed~~, `*fixed*`."},
		{"whitespace", `<p>A<strong> bold </strong>B<em> italic </em>C<del> gone </del>D</p>`,
			"A **bold** B *italic* C ~~gone~~ D"},
		{"literal", "<p>*literal* _literal_ [not a link] &lt;b&gt; &amp;copy; C:\\tmp</p>",
			`\*literal\* \_literal\_ \[not a link\] \<b\> &amp;copy; C:\\tmp`},
		{"markers", "<p>1. Not a list</p><p>- Not a list</p><p># Not a heading</p><p>---</p>",
			"1\\. Not a list\n\n\\- Not a list\n\n\\# Not a heading\n\n\\-\\-\\-"},
		{"breaks", "<p>First<br>Second</p><p>Third</p>", "First  \nSecond\n\nThird"},
		{"lists", `<ol start="3"><li>Third<ul><li>Nested</li></ul></li><li>Fourth</li></ol>`,
			"3. Third\n   - Nested\n4. Fourth"},
		{"quote", `<blockquote><p>First</p><p>Second</p></blockquote>`, "> First\n> \n> Second"},
		{"rule", `<p>Before</p><hr><p>After</p>`, "Before\n\n---\n\nAfter"},
		{"code", "<pre><code class=\"language-go\">a := `x`\n  b := 2</code></pre>", "```\na := `x`\n  b := 2\n```"},
		{"code spaces", `<code> </code>`, "` `"},
		{"empty code", `<code></code>`, "<code></code>"},
		{"link", `<a href="https://example.com/a(b)?q=a b">A [label]</a>`,
			`[A \[label\]](https://example.com/a%28b%29?q=a%20b)`},
		{"superscript", `<p>x<sup>2</sup> and H<sub>2</sub>O</p>`, "x<sup>2</sup> and H<sub>2</sub>O"},
		{"underline literal", `<u>*not bold*</u>`, `<u>\*not bold\*</u>`},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := Convert(test.source)
			if err != nil {
				t.Fatal(err)
			}
			if got.Markdown != test.want {
				t.Errorf("Markdown = %q, want %q", got.Markdown, test.want)
			}
		})
	}
}
