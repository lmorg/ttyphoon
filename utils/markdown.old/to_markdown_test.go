package markdown_test

import (
	"strings"
	"testing"

	"github.com/lmorg/mxtty/utils/markdown"
)

func TestParseToMarkdown(t *testing.T) {
	tests := []string{
		"# Heading\n\nParagraph",
		/*"```\nCode\n    Block\n```",
		"```pseudocode\nCode\n    Block\n```",
		"hello `world`!",
		"Lets\n\n> quote\n> something cool\n\nok!",
		"1\n2\n\n3",*/
	}

	for i, s := range tests {
		n := markdown.Parse([]rune(s))
		md := strings.TrimSpace(n.Print(markdown.OUTPUT_FORMAT_MARKDOWN))
		if s != md {
			t.Errorf("Error in test %d", i)
			//t.Logf("  Input:\n%s", s)
			t.Logf("  Expected:\n%s", s)
			t.Logf("  Actual:\n%s", md)
			t.Logf("  Expected chars:\n%v", []rune(s))
			t.Logf("  Actual chars:\n%v", []rune(md))
			t.Logf("  Json:\n%s", n.String())
		}
	}
}
