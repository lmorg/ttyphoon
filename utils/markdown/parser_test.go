package markdown

import (
	"os"
	"strings"
	"testing"
)

func TestParse_Headings(t *testing.T) {
	md := "# Heading 1\n## Heading 2\n### Heading 3"
	ast := Parse(md)
	if len(ast.Children) != 3 {
		t.Fatalf("expected 3 headings, got %d", len(ast.Children))
	}
	for i, want := range []string{"Heading 1", "Heading 2", "Heading 3"} {
		h := ast.Children[i]
		if h.Type != NodeHeading {
			t.Errorf("child %d: expected NodeHeading, got %v", i, h.Type)
		}
		if h.Children[0].Children[0].Text != want {
			t.Errorf("child %d: expected text %q, got %q", i, want, h.Children[0].Children[0].Text)
		}
	}
}

func TestParse_ParagraphsAndInlines(t *testing.T) {
	md := "This is **bold** and *italic* and `code`."
	ast := Parse(md)
	if len(ast.Children) != 1 {
		t.Fatalf("expected 1 paragraph, got %d", len(ast.Children))
	}
	para := ast.Children[0]
	if para.Type != NodeParagraph {
		t.Fatalf("expected NodeParagraph, got %v", para.Type)
	}
	// Check for bold, italic, code inline nodes
	foundBold := false
	foundItalic := false
	foundCode := false
	var textConcat strings.Builder
	var walk func(n *Node)
	walk = func(n *Node) {
		switch n.Type {
		case NodeBold:
			foundBold = true
		case NodeItalic:
			foundItalic = true
		case NodeCodeInline:
			foundCode = true
		case NodeText:
			textConcat.WriteString(n.Text)
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(para)
	if !foundBold || !foundItalic || !foundCode {
		t.Errorf("expected all inline types, got bold=%v italic=%v code=%v", foundBold, foundItalic, foundCode)
	}
	if !strings.Contains(textConcat.String(), "This is ") {
		t.Errorf("expected text node with 'This is ', got %q", textConcat.String())
	}
}

func TestParse_List(t *testing.T) {
	md := "- Item 1\n- Item 2\n  - Nested"
	ast := Parse(md)
	if len(ast.Children) == 0 || ast.Children[0].Type != NodeListBullet {
		t.Fatalf("expected NodeListBullet at root")
	}
	list := ast.Children[0]
	if len(list.Children) != 2 {
		t.Fatalf("expected 2 list items, got %d", len(list.Children))
	}
	if list.Children[1].Children[1].Type != NodeListBullet {
		t.Errorf("expected nested NodeListBullet, got %v", list.Children[1].Children[1].Type)
	}
}

func TestParse_Table(t *testing.T) {
	md := "| H1 | H2 |\n|----|----|\n| C1 | C2 |"
	ast := Parse(md)
	found := false
	for _, n := range ast.Children {
		if n.Type == NodeTable {
			found = true
			if len(n.Children) != 3 {
				t.Errorf("expected 3 table rows, got %d", len(n.Children))
			}
		}
	}
	if !found {
		t.Errorf("expected NodeTable in AST")
	}
}

func TestParse_CodeBlock(t *testing.T) {
	md := "```\ncode block\n```"
	ast := Parse(md)
	found := false
	for _, n := range ast.Children {
		if n.Type == NodeCodeBlock {
			found = true
			if !strings.Contains(n.Text, "code block") {
				t.Errorf("expected code block text, got %q", n.Text)
			}
		}
	}
	if !found {
		t.Errorf("expected NodeCodeBlock in AST")
	}
}

func TestParse_SampleFile(t *testing.T) {
	b, err := os.ReadFile("testdata/sample.md")
	if err != nil {
		t.Fatalf("failed to read sample.md: %v", err)
	}
	ast := Parse(string(b))
	if ast == nil || len(ast.Children) == 0 {
		t.Errorf("expected non-empty AST for sample.md")
	}
}
