package markdownutil

import (
	"strings"
	"testing"
	"text/template"
)

func TestEscapeMarkdownText(t *testing.T) {
	got := EscapeMarkdownText("<b>bold</b> **text** | [link](url)\nnext")
	want := "&lt;b&gt;bold&lt;/b&gt; \\*\\*text\\*\\* \\| \\[link\\]\\(url\\) next"
	if got != want {
		t.Fatalf("EscapeMarkdownText() = %q, want %q", got, want)
	}
}

func TestMarkdownCodeBlockUsesFenceLongerThanInput(t *testing.T) {
	input := "echo ```\necho `````"
	got := MarkdownCodeBlock(input, "bash")
	want := "``````bash\n" + input + "\n``````"
	if got != want {
		t.Fatalf("MarkdownCodeBlock() = %q, want %q", got, want)
	}
}

func TestMarkdownTableCodeKeepsMultilineValueInOneCell(t *testing.T) {
	got := MarkdownTableCode("+a | <b>\r\n-`c`")
	want := "<pre><code>+a &#124; &lt;b&gt;&#10;-&#96;c&#96;</code></pre>"
	if got != want {
		t.Fatalf("MarkdownTableCode() = %q, want %q", got, want)
	}
}

func TestTemplateFuncMap(t *testing.T) {
	tmpl, err := template.New("description").Funcs(TemplateFuncMap()).Parse("{{ escapeMarkdown . }}\n{{ markdownCodeBlock . }}")
	if err != nil {
		t.Fatalf("Parse(): %v", err)
	}

	var output strings.Builder
	if err := tmpl.Execute(&output, "<unsafe> ```"); err != nil {
		t.Fatalf("Execute(): %v", err)
	}
	if !strings.Contains(output.String(), "&lt;unsafe&gt;") || !strings.Contains(output.String(), "````\n<unsafe> ```\n````") {
		t.Fatalf("template output = %q, want escaped text and a safe code fence", output.String())
	}
}
